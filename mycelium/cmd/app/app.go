package main

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"mycelium/internal/crawler"
	"mycelium/internal/redis"
)

type Environment struct {
	RedisAddr       string
	RedisPass       string
	RedisDB         int
	FilestoreOutDir string
}

type MyceliumConfig struct {
	seedFile    string
	agentsFile  string
	proxyFile   string
	numCrawlers int
}

type Mycelium struct {
	config  MyceliumConfig
	cache   redis.RedisCache
	crawler crawler.Crawler
}

func (app *Mycelium) seed(ctx context.Context) {
	var seed []crawler.QueueItem

	urls, err := initSeedUrls(app.config.seedFile)
	if err != nil {
		panic(err)
	}

	for _, seedUrl := range urls {
		seed = append(seed, redis.NewQueueItem(seedUrl))
	}

	err = app.crawler.Seed(ctx, seed)
	if err != nil {
		panic(err)
	}
}

func (app *Mycelium) crawl(ctx context.Context) {
	var wg sync.WaitGroup

	makeQueueItem := func(u *url.URL) crawler.QueueItem {
		return redis.NewQueueItem(u)
	}

	crawlRoutine := func(wg *sync.WaitGroup, i int) {
		defer wg.Done()
		err := app.crawler.Crawl(ctx, makeQueueItem)
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
