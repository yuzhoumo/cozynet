package main

import (
	"context"
	"fmt"

	"mycelium/internal/crawler"
)

type MyceliumConfig struct {
	seedFilePath   string
	agentsFilePath string
	proxyFilePath  string
}

type Mycelium struct {
	config MyceliumConfig
}

func main() {
	var app Mycelium

	conf := app.config
	initCliFlags(&conf)

	urls, err := initSeedUrls(conf.seedFilePath)
	if err != nil {
		panic(err)
	}

	var options []crawler.CrawlerOption

	proxyChooser, err := initProxyChooser(conf.proxyFilePath)
	if err != nil {
		panic(err)
	} else if proxyChooser != nil {
		options = append(options, crawler.WithProxyChooser(proxyChooser))
	}

	userAgentChooser, err := initUserAgentChooser(conf.agentsFilePath)
	if err != nil {
		panic(err)
	} else if userAgentChooser != nil {
		options = append(options, crawler.WithUserAgentChooser(userAgentChooser))
	}

	ctx := context.Background()
	crawl := crawler.NewCrawler(options...)

	for i := 0; i < 10; i++ {
		fmt.Printf("Requesting %s\n", urls[i].String())
		_, err := crawl.GetPageContent(ctx, urls[i])
		if err != nil {
			panic(err)
		}
	}
}
