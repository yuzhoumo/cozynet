package main

import (
	"context"
	"fmt"
	"sync"

	"mycelium/internal/cache"
	"mycelium/internal/crawler"
)

type Environment struct {
	RedisAddr            string
	RedisPass            string
	RedisDB              int
	FilestoreOutDir      string
	FungicideQueueKey    string
	MyceliumIngressKey   string
	MyceliumBlacklistKey string
}

type MyceliumConfig struct {
	seedFile            string
	agentsFile          string
	proxyFile           string
	domainBlacklistFile string
	numCrawlers         int
	maxIdleSeconds      int
}

type Mycelium struct {
	config  MyceliumConfig
	cache   cache.CrawlerCache
	crawler crawler.Crawler
}

func (app *Mycelium) seed(ctx context.Context) {
	urls, err := initSeedUrls(app.config.seedFile)
	if err != nil {
		panic(err)
	}

	var seed []string
	for _, seedUrl := range urls {
		seed = append(seed, seedUrl.String())
	}

	err = app.crawler.Seed(ctx, seed)
	if err != nil {
		panic(err)
	}
}

func (app *Mycelium) crawl(ctx context.Context) {
	var wg sync.WaitGroup

	crawlRoutine := func(wg *sync.WaitGroup, i int) {
		defer wg.Done()
		fmt.Printf("Crawler %d starting\n", i)
		err := app.crawler.Crawl(ctx)
		if err != nil {
			panic(fmt.Errorf("crawler %d failed with error: %w", i, err))
		}
	}

	wg.Add(app.config.numCrawlers)
	for i := 0; i < app.config.numCrawlers; i++ {
		go crawlRoutine(&wg, i)
	}

	wg.Wait()
}
