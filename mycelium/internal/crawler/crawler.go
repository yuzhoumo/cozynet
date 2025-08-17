package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type StoreItem interface {
	Marshal() ([]byte, error)
}

type Store interface {
	Store(item StoreItem, extension string) (id string, err error)
	Retrieve(id string, extension string) (data []byte, err error)
}

type QueueItem interface {
	GetLocation() string
	GetRetries() int32
	SetRetries(retries int32)
	ProtoReflect() protoreflect.Message
}

type CrawlerCache interface {
	QueuePush(context.Context, QueueItem) error
	QueuePop(context.Context) (QueueItem, error)
	QueueSize(context.Context) (int32, error)
	Visit(context.Context, QueueItem) error
	IsVisited(context.Context, QueueItem) (bool, error)
}

type StringChooser interface {
	Pick() string
}

type Crawler struct {
	client           *http.Client
	userAgentChooser StringChooser
	proxyChooser     StringChooser
	cache            CrawlerCache
	store            Store
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

func (c *Crawler) Seed(ctx context.Context, seed []QueueItem) error {
	size, err := c.cache.QueueSize(ctx)
	if err != nil {
		return fmt.Errorf("failed to get queue size: %w", err)
	}

	if size > 0 {
		fmt.Printf("Queue is non-empty length %d, skipping seed stage\n", size)
		return nil
	}

	for _, item := range seed {
		err := c.cache.QueuePush(ctx, item)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Crawler) Crawl(ctx context.Context, makeQueueItem func(*url.URL) QueueItem) error {
	size, err := c.cache.QueueSize(ctx)
	if err != nil {
		return err
	}

	for size > 0 {
		curr, err := c.cache.QueuePop(ctx)
		if err != nil {
			return err
		}

		if curr.GetRetries() > maxRetries {
			continue
		}

		isVisited, err := c.cache.IsVisited(ctx, curr)
		if err != nil {
			fmt.Printf("failed to check if %v is visited: %s\n", curr, err.Error())
			curr.SetRetries(curr.GetRetries() + 1)
			c.cache.QueuePush(ctx, curr)
			continue
		} else if isVisited {
			continue
		} else {
			c.cache.Visit(ctx, curr)
		}

		size, err = c.cache.QueueSize(ctx)
		if err != nil {
			return err
		}

		parsedUrl, err := url.Parse(curr.GetLocation())
		if err != nil {
			fmt.Printf("malformed url: %s", curr.GetLocation())
			continue
		}

		page, err := c.GetPage(ctx, parsedUrl)
		if err != nil {
			fmt.Printf("failed to get page %s: %s\n", curr.GetLocation(), err.Error())
			continue
		}

		_, err = c.store.Store(page, ".json") // TODO: record page id in db
		if err != nil {
			fmt.Printf("failed to store page: %s\n", err.Error())
		}

		for _, neighbor := range page.Links {
			c.cache.QueuePush(ctx, makeQueueItem(&neighbor))
		}
	}

	return nil
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
		fmt.Printf("[%s] %d outlinks\n", page.Location, len(page.Links))
	} else {
		fmt.Println("TODO: PARSE PLAINTEXT PAGE")
	}

	return page, nil
}

func proxyURL(proxyChooser StringChooser) func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		return url.Parse(proxyChooser.Pick())
	}
}
