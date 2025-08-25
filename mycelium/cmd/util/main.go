package main

import (
	"context"
	"flag"
	"net/url"
	"os"

	"mycelium/internal/crawler"
)

func main() {
	var location string
	var output string

	flag.StringVar(&location, "url", "", "url to crawl")
	flag.StringVar(&output, "out", "./out.json", "output file")
	flag.Parse()

	parsedUrl, err := url.Parse(location)
	if err != nil {
		panic(err)
	}

	c := *crawler.NewCrawler(nil, nil)

	page, err := c.GetPage(context.Background(), parsedUrl)
	if err != nil {
		panic(err)
	}

	data, err := page.Marshal()
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(output, data, 0755)
	if err != nil {
		panic(err)
	}
}
