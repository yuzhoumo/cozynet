package crawler

import (
    "fmt"
    "net/url"
    "io"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Page struct {
    Title       string
    Description string
    Headers     []string
    Content     []string
    Links       []url.URL
    Location    *url.URL
    ScriptLinks []url.URL
}

func NewPage(loc *url.URL) *Page {
    return &Page{ Location: loc }
}

func (p *Page) NormalizePageURL(loc string) (*url.URL, error) {
    parsedUrl, err := url.Parse(loc)
    if err != nil {
        return nil, fmt.Errorf("failed to parse url: %w", err)
    }

    if parsedUrl.Hostname() != "" {
        return parsedUrl, nil
    }

    joined, err := url.JoinPath(p.Location.String(), parsedUrl.String())
    if err != nil {
        return nil, fmt.Errorf("failed to normalize url: %w", err)
    }

    joinedParsed, err := url.Parse(joined)
    if err != nil {
        return nil, fmt.Errorf("failed to parse normalized url: %w", err)
    }

    return joinedParsed, nil
}

func (p *Page) ParseHtmlPage(r io.Reader) {
	tokenizer := html.NewTokenizer(r)
	for tokenizer.Err() == nil {
        p.parseHtmlToken(tokenizer)
        tokenizer.Next()
	}
}

func (p *Page) parseHtmlToken(tokenizer *html.Tokenizer) {
    t := tokenizer.Token()
    switch t.DataAtom {
    case atom.A:
        p.parseHtmlLink(&t)
    case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
        p.parseHtmlHeader(&t)
    default:
        fmt.Println("placeholder parse html token")
    }
}

func (p *Page) parseHtmlHeader(t *html.Token) {
    p.Headers = append(p.Headers, t.Data)
}

func (p *Page) parseHtmlLink(t *html.Token) {
    for _, a := range t.Attr {
        if a.Key != "href" {
            continue
        }

        normalizedUrl, err := p.NormalizePageURL(a.Val)
        if err != nil {
            fmt.Printf("error normalizing url: %v", err)
        }

        p.Links = append(p.Links, *normalizedUrl)
    }
}
