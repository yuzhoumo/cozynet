package crawler

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type StringChooser interface {
	Pick() string
}

type Crawler struct {
	client           http.Client
	proxyChooser     StringChooser
	userAgentChooser StringChooser
}

func NewCrawler(optClient *http.Client, proxyChooser StringChooser, userAgentChooser StringChooser) *Crawler {
	client := optClient

	if client == nil {
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: proxyURL(proxyChooser),
			},
		}
	}

	return &Crawler{
		client:           *client,
		proxyChooser:     proxyChooser,
		userAgentChooser: userAgentChooser,
	}
}

func (r *Crawler) GetPageContent(ctx context.Context, url *url.URL) (*string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := r.userAgentChooser.Pick()
	req.Header.Set(userAgentCanonicalHeader, userAgent)

	fmt.Printf("set user agent: %s\n", userAgent)

	res, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request %s: %w", url.String(), err)
	}
	defer res.Body.Close()

	contentType := res.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/") {
		return nil, fmt.Errorf("page content %s was not type 'text', got: %s", url.String(), contentType)
	}

	if strings.HasPrefix(contentType, "text/html") {
		parseHtml(res.Body)
	} else {
		parsePlaintext(res.Body)
	}

	return nil, nil
}

func proxyURL(proxyChooser StringChooser) func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		return url.Parse(proxyChooser.Pick())
	}
}

func parseHtml(r io.Reader) {
	tokenizer := html.NewTokenizer(r)
	for tokenizer.Err() == nil {
		t := tokenizer.Token()
		if t.DataAtom == atom.A {
			for _, a := range t.Attr {
				if a.Key == "href" {
					fmt.Println(a.Val)
				}
			}
		}
		tokenizer.Next()
	}
}

func parsePlaintext(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}
