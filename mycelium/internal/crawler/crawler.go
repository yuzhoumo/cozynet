package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type QueueItem interface {
	GetLocation() string
	GetRetries() int32
	SetRetries(int32)
	ProtoReflect() protoreflect.Message
}

type Visited interface {
	Visit(context.Context, QueueItem) error
	IsVisited(context.Context, QueueItem) (bool, error)
	Reset(context.Context) error
}

type Queue interface {
	Enqueue(context.Context, QueueItem) error
	Pop(context.Context) (QueueItem, error)
	Size(context.Context) (int32, error)
}

type StringChooser interface {
	Pick() string
}

type Crawler struct {
	client           *http.Client
	userAgentChooser StringChooser
	proxyChooser     StringChooser
	queue            Queue
	visited          Visited
}

type CrawlerOption func(*Crawler)

func NewCrawler(queue Queue, visited Visited, opt ...CrawlerOption) *Crawler {
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

	c.queue = queue

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

func (c *Crawler) Crawl(ctx context.Context, seed []QueueItem, makeQueueItem func(*url.URL) QueueItem) error {
	for _, item := range seed {
		c.queue.Enqueue(ctx, item)
	}

	size, err := c.queue.Size(ctx)
	if err != nil {
		return err
	}

	for size > 0 {
		curr, err := c.queue.Pop(ctx)
		if err != nil {
			return err
		}

		if curr.GetRetries() > maxRetries {
			continue
		}

		isVisited, err := c.visited.IsVisited(ctx, curr)
		if err != nil {
			fmt.Printf("failed to check if %v is visited: %s", curr, err.Error())
			curr.SetRetries(curr.GetRetries() + 1)
			c.queue.Enqueue(ctx, curr)
			continue
		} else if isVisited {
			continue
		} else {
			c.visited.Visit(ctx, curr)
		}

		size, err = c.queue.Size(ctx)
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
			fmt.Printf("failed to get page %s: %s", curr.GetLocation(), err.Error())
			continue
		}

		for _, neighbor := range page.Links {
			c.queue.Enqueue(ctx, makeQueueItem(&neighbor))
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

	fmt.Printf("set user agent: %s\n", userAgent)

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
		fmt.Println(page.String())
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
