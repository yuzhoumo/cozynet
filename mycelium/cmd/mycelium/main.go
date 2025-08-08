package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"mycelium/internal/crawler"
)

func main() {
	var seedFilePath string
	var agentsFilePath string

	flag.StringVar(&seedFilePath, "seedfile", "./seed.txt", "newline delimited list of seed urls")
	flag.StringVar(&agentsFilePath, "agentsfile", "./agents.json", "user agents json")
	flag.Parse()

	urls, err := initSeedUrls(seedFilePath)
	if err != nil {
		panic(err)
	}

	fmt.Printf("successfully parsed %d urls\n", len(urls))

	userAgentChooser, err := initUserAgentChooser(agentsFilePath)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	req := crawler.NewCrawler(&http.Client{}, userAgentChooser)

	for i := 0; i < 10; i++ {
		fmt.Printf("Requesting %s\n", urls[i].String())
		req.GetPageContent(ctx, urls[i])
	}
}
