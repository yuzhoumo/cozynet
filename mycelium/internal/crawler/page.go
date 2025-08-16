package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Page struct {
	Title         string
	Description   string
	Author        string
	Keywords      []string
	Headings      []string
	Content       []string
	Links         []url.URL
	ScriptLinks   []url.URL
	ScriptContent []string
	Location      *url.URL
}

func NewPage(loc *url.URL) *Page {
	return &Page{Location: loc}
}

func urlsToStrings(urls []url.URL) []string {
	var res []string
	for _, u := range urls {
		res = append(res, u.String())
	}
	return res
}

func (p *Page) Marshal() ([]byte, error) {
	return json.Marshal(struct {
		Title         string   `json:"title"`
		Description   string   `json:"description"`
		Author        string   `json:"author"`
		Keywords      []string `json:"keywords"`
		Headings      []string `json:"headings"`
		Content       []string `json:"content"`
		Links         []string `json:"links"`
		ScriptLinks   []string `json:"script_links"`
		ScriptContent []string `json:"script_content"`
		Location      string   `json:"location"`
		CreatedAt     int64    `json:"created_at"`
	}{
		Title:         p.Title,
		Description:   p.Description,
		Author:        p.Author,
		Keywords:      p.Keywords,
		Headings:      p.Headings,
		Content:       p.Content,
		Links:         urlsToStrings(p.Links),
		ScriptLinks:   urlsToStrings(p.ScriptLinks),
		ScriptContent: p.ScriptContent,
		Location:      p.Location.String(),
		CreatedAt:     time.Now().UnixMilli(),
	})
}

func (p *Page) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "\nPAGE: %s\n", p.Location)
	b.WriteString(strings.Repeat("-", 30) + "\n")

	fmt.Fprintf(&b, "Title: %s\n", p.Title)
	fmt.Fprintf(&b, "Description: %s\n", p.Description)
	fmt.Fprintf(&b, "Author: %s\n", p.Author)

	if len(p.Keywords) > 0 {
		b.WriteString("Keywords:\n")
		for _, k := range p.Keywords {
			fmt.Fprintf(&b, "  - %s\n", k)
		}
	}

	if len(p.Headings) > 0 {
		b.WriteString("Headings:\n")
		for _, h := range p.Headings {
			fmt.Fprintf(&b, "  - %s\n", h)
		}
	}

	if len(p.Content) > 0 {
		b.WriteString("Content:\n")
		for _, c := range p.Content {
			fmt.Fprintf(&b, "  - %s\n", c)
		}
	}

	if len(p.Links) > 0 {
		b.WriteString("Links:\n")
		for _, link := range p.Links {
			fmt.Fprintf(&b, "  - %s\n", link.String())
		}
	}

	if len(p.ScriptLinks) > 0 {
		b.WriteString("Script Links:\n")
		for _, sl := range p.ScriptLinks {
			fmt.Fprintf(&b, "  - %s\n", sl.String())
		}
	}

	if len(p.ScriptContent) > 0 {
		b.WriteString("Script Content:\n")
		for i, sc := range p.ScriptContent {
			fmt.Fprintf(&b, "  [%d] %s\n", i+1, sc)
		}
	}

	b.WriteString(strings.Repeat("-", 30) + "\n")
	return b.String()
}

func (p *Page) NormalizePageURL(loc string) (*url.URL, error) {
	trimmed := strings.TrimSpace(loc)
	parsedUrl, err := url.Parse(trimmed)
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

	var tag atom.Atom
	for tokenizer.Err() == nil {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			break
		case html.StartTagToken:
			t := tokenizer.Token()
			tag = t.DataAtom
			p.parseHtmlTagToken(&t, tag)
		case html.TextToken:
			t := tokenizer.Token()
			p.parseHtmlTextToken(&t, tag)
		}
	}
}

func (p *Page) parseHtmlTagToken(token *html.Token, tag atom.Atom) {
	switch tag {
	case atom.A:
		p.parseHtmlLink(token)
	case atom.Script:
		p.parseHtmlScriptAttributes(token)
	case atom.Meta:
		p.parseHtmlMeta(token)
	}
}

func (p *Page) parseHtmlTextToken(token *html.Token, tag atom.Atom) {
	switch tag {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		p.parseHtmlHeadings(token)
	case atom.Title:
		p.parseHtmlTitle(token)
	case atom.Script:
		p.parseHtmlScriptContent(token)
	case atom.P, atom.Span, atom.Pre, atom.Code,
		atom.Em, atom.Strong, atom.B, atom.I, atom.Mark, atom.Small,
		atom.Abbr, atom.Cite, atom.Q, atom.Blockquote, atom.Kbd, atom.Samp,
		atom.Var, atom.Li, atom.Dt, atom.Dd, atom.Th, atom.Td, atom.Caption:
		p.parseContent(token)
	}
}

func (p *Page) parseContent(t *html.Token) {
	trimmed := strings.TrimSpace(t.Data)
	if trimmed != "" {
		p.Content = append(p.Content, trimmed)
	}
}

func (p *Page) parseHtmlTitle(t *html.Token) {
	trimmed := strings.TrimSpace(t.Data)
	if trimmed != "" {
		p.Title = trimmed
	}
}

func (p *Page) parseHtmlHeadings(t *html.Token) {
	trimmed := strings.TrimSpace(t.Data)
	if trimmed != "" {
		p.Headings = append(p.Headings, trimmed)
	}
}

func (p *Page) parseHtmlLink(t *html.Token) {
	for _, a := range t.Attr {
		if a.Key != "href" {
			continue
		}

		normalizedUrl, err := p.NormalizePageURL(a.Val)
		if err != nil {
			fmt.Printf("error normalizing url: %v", err)
			continue
		}

		p.Links = append(p.Links, *normalizedUrl)
	}
}

func (p *Page) parseHtmlMeta(t *html.Token) {
	var content string
	var name string

	for _, a := range t.Attr {
		switch a.Key {
		case "name":
			name = strings.TrimSpace(a.Val)
		case "content":
			content = strings.TrimSpace(a.Val)
		}
	}

	if content == "" {
		return
	}

	switch name {
	case "description":
		p.Description = content
	case "keywords":
		p.Keywords = strings.Split(content, ",")
	case "author":
		p.Author = content
	}
}

func (p *Page) parseHtmlScriptContent(t *html.Token) {
	trimmed := strings.TrimSpace(t.Data)
	if trimmed != "" {
		p.ScriptContent = append(p.ScriptContent, trimmed)
	}
}

func (p *Page) parseHtmlScriptAttributes(t *html.Token) {
	for _, a := range t.Attr {
		if a.Key != "src" {
			continue
		}

		normalizedUrl, err := p.NormalizePageURL(a.Val)
		if err != nil {
			fmt.Printf("error normalizing url: %v", err)
			continue
		}

		p.ScriptLinks = append(p.ScriptLinks, *normalizedUrl)
	}
}
