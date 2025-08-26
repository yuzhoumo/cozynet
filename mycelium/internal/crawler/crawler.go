package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type StoreItem interface {
	Prefix() string
	Marshal() ([]byte, error)
}

type Store interface {
	Store(item StoreItem, extension string) (id string, err error)
	Retrieve(id string, extension string) (data []byte, err error)
}

type UrlFilter interface {
	Filter(loc *url.URL) bool
}

type IngressItem struct {
	Location string `json:"location"`
	Retries  int32  `json:"retries"`
}

type CrawlerCache interface {
	Visit(context.Context, string) error
	IsVisited(context.Context, string) (bool, error)
	PushToFungicide(context.Context, string, string) error
	PushToMyceliumIngress(context.Context, string, string) error
	PopFromMyceliumIngress(context.Context, string) (string, error)
	IsBlacklisted(context.Context, string, string) (bool, error)
	IngressQueueSize(context.Context, string) (int32, error)
}

type StringChooser interface {
	Pick() string
}

type Crawler struct {
	client               *http.Client
	userAgentChooser     StringChooser
	proxyChooser         StringChooser
	cache                CrawlerCache
	store                Store
	urlFilters           []UrlFilter
	maxIdleSeconds       int
	idleSeconds          int
	fungicideQueueKey    string
	myceliumIngressKey   string
	myceliumBlacklistKey string
}

type CrawlerOption func(*Crawler)

func NewCrawler(cache CrawlerCache, store Store, opt ...CrawlerOption) *Crawler {
	c := new(Crawler)
	for _, o := range opt {
		o(c)
	}

	if c.client == nil {
		c.client = &http.Client{}
	}

	if c.proxyChooser != nil {
		c.client.Transport = &http.Transport{
			Proxy: proxyURL(c.proxyChooser),
		}
	}

	c.cache = cache
	c.store = store

	return c
}

func WithUrlFilters(filters []UrlFilter) CrawlerOption {
	return func(c *Crawler) {
		c.urlFilters = filters
	}
}

func WithMaxIdle(maxIdleSeconds int) CrawlerOption {
	return func(c *Crawler) {
		c.maxIdleSeconds = maxIdleSeconds
	}
}

func WithHttpClient(client *http.Client) CrawlerOption {
	return func(c *Crawler) {
		c.client = client
	}
}

func WithProxyChooser(proxyChooser StringChooser) CrawlerOption {
	return func(c *Crawler) {
		c.proxyChooser = proxyChooser
	}
}

func WithUserAgentChooser(userAgentChooser StringChooser) CrawlerOption {
	return func(c *Crawler) {
		c.userAgentChooser = userAgentChooser
	}
}

func WithFungicideQueueKey(key string) CrawlerOption {
	return func(c *Crawler) {
		c.fungicideQueueKey = key
	}
}

func WithMyceliumIngressKey(key string) CrawlerOption {
	return func(c *Crawler) {
		c.myceliumIngressKey = key
	}
}

func WithMyceliumBlacklistKey(key string) CrawlerOption {
	return func(c *Crawler) {
		c.myceliumBlacklistKey = key
	}
}

func (c *Crawler) Seed(ctx context.Context, seed []string) error {
	if c.myceliumIngressKey == "" {
		return fmt.Errorf("mycelium ingress queue key not configured")
	}

	size, err := c.cache.IngressQueueSize(ctx, c.myceliumIngressKey)
	if err != nil {
		return fmt.Errorf("failed to get ingress queue size: %w", err)
	}

	if size > 0 {
		fmt.Printf("Ingress queue is non-empty (length %d), skipping seed stage\n", size)
		return nil
	}

	for _, seedUrl := range seed {
		ingressItem := IngressItem{
			Location: seedUrl,
			Retries:  0,
		}

		itemJSON, err := json.Marshal(ingressItem)
		if err != nil {
			return fmt.Errorf("failed to marshal seed item: %w", err)
		}

		err = c.cache.PushToMyceliumIngress(ctx, string(itemJSON), c.myceliumIngressKey)
		if err != nil {
			return fmt.Errorf("failed to seed %s: %w", seedUrl, err)
		}
	}

	fmt.Printf("Seeded %d URLs to ingress queue\n", len(seed))
	return nil
}

func (c *Crawler) Crawl(ctx context.Context) error {
	if c.myceliumIngressKey == "" {
		return fmt.Errorf("mycelium ingress queue key not configured")
	}

	fmt.Printf("Crawler starting, waiting for items from ingress queue...\n")

	for {
		incomingJSON, err := c.cache.PopFromMyceliumIngress(ctx, c.myceliumIngressKey)
		if err != nil {
			// Handle "no items available" case - continue polling
			if err.Error() == "no items available in queue" {
				continue
			}
			// For other errors, log and continue (with brief delay to avoid spam)
			fmt.Printf("Error popping from ingress queue: %s\n", err.Error())
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
				continue
			}
		}

		var curr IngressItem
		if err := json.Unmarshal([]byte(incomingJSON), &curr); err != nil {
			fmt.Printf("failed to parse incoming JSON: %s\n", err.Error())
			continue
		}

		if curr.Retries > maxRetries {
			continue
		}

		isVisited, err := c.cache.IsVisited(ctx, curr.Location)
		if err != nil {
			fmt.Printf("failed to check if %s is visited: %s\n", curr.Location, err.Error())
			curr.Retries = curr.Retries + 1
			retryJSON, _ := json.Marshal(curr)
			c.cache.PushToMyceliumIngress(ctx, string(retryJSON), c.myceliumIngressKey)
			continue
		} else if isVisited {
			continue
		} else {
			c.cache.Visit(ctx, curr.Location)
		}

		parsedUrl, err := url.Parse(curr.Location)
		if err != nil {
			fmt.Printf("malformed url: %s", curr.Location)
			continue
		}

		if c.filter(parsedUrl) {
			fmt.Printf("[BLOCKED] url: %s\n", curr.Location)
			continue
		}

		// Check domain blacklist from fungicide
		if c.myceliumBlacklistKey != "" {
			isBlacklisted, err := c.cache.IsBlacklisted(ctx, parsedUrl.Hostname(), c.myceliumBlacklistKey)
			if err != nil {
				fmt.Printf("failed to check blacklist for %s: %s\n", parsedUrl.Hostname(), err.Error())
			} else if isBlacklisted {
				fmt.Printf("[BLACKLISTED] %s\n", curr.Location)
				continue
			}
		}

		page, err := c.GetPage(ctx, parsedUrl)
		if err != nil {
			fmt.Printf("failed to get page %s: %s\n", curr.Location, err.Error())
			continue
		}

		// Send page to fungicide for classification instead of storing to file
		if c.fungicideQueueKey != "" {
			pageJSON, err := page.Marshal()
			if err != nil {
				fmt.Printf("failed to marshal page %s: %s\n", curr.Location, err.Error())
				continue
			}

			err = c.cache.PushToFungicide(ctx, string(pageJSON), c.fungicideQueueKey)
			if err != nil {
				fmt.Printf("failed to push page to fungicide %s: %s\n", curr.Location, err.Error())
				continue
			}

			fmt.Printf("[SENT TO FUNGICIDE] %s\n", curr.Location)
		} else {
			// Fallback to file storage if fungicide not configured
			_, err = c.store.Store(page, ".json")
			if err != nil {
				fmt.Printf("failed to store page: %s\n", err.Error())
			}

			// Direct link queuing only if not using fungicide - queue back to ingress
			for _, neighbor := range page.Links {
				neighborItem := IngressItem{
					Location: neighbor.String(),
					Retries:  0,
				}
				neighborJSON, _ := json.Marshal(neighborItem)
				c.cache.PushToMyceliumIngress(ctx, string(neighborJSON), c.myceliumIngressKey)
			}
		}
	}
}

func (c *Crawler) filter(loc *url.URL) bool {
	for _, filter := range c.urlFilters {
		if filter.Filter(loc) {
			return true
		}
	}
	return false
}

func (r *Crawler) GetPage(ctx context.Context, loc *url.URL) (*Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, loc.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := defaultUserAgent
	if r.userAgentChooser != nil {
		userAgent = r.userAgentChooser.Pick()
	}
	req.Header.Set(userAgentCanonicalHeader, userAgent)

	res, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request %s: %w", loc.String(), err)
	}
	defer res.Body.Close()

	contentType := res.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/") {
		return nil, fmt.Errorf("page content %s was not type 'text', got: %s", loc.String(), contentType)
	}

	page := NewPage(loc)

	if strings.HasPrefix(contentType, "text/html") {
		page.ParseHtmlPage(res.Body)
	} else {
		fmt.Println("Skipping non text/html page.")
	}

	return page, nil
}

func proxyURL(proxyChooser StringChooser) func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		return url.Parse(proxyChooser.Pick())
	}
}
