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

	proxyChooser, err := initProxyChooser(conf.proxyFilePath)
	if err != nil {
		panic(err)
	}

	userAgentChooser, err := initUserAgentChooser(conf.agentsFilePath)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	req := crawler.NewCrawler(nil, proxyChooser, userAgentChooser)

	for i := 0; i < 10; i++ {
		fmt.Printf("Requesting %s\n", urls[i].String())
		req.GetPageContent(ctx, urls[i])
	}
}
