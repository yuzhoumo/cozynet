package main

import (
	"context"
	"os"

	"mycelium/internal/crawler"
	"mycelium/internal/redis"
	"mycelium/internal/store"
)

func main() {
	var app Mycelium
	var env Environment

	ctx := context.Background()

	initCliFlags(&app.config)
	if err := initEnvironment(&env); err != nil {
		panic(err)
	}

	// create redis cache
	cache, err := redis.NewRedisCache(ctx, &redis.RedisCacheOptions{
		Addr: env.RedisAddr,
		Pass: env.RedisPass,
		DB:   env.RedisDB,
	})
	if err != nil {
		panic(err)
	}
	app.cache = *cache

	// create crawler
	var options []crawler.CrawlerOption
	proxyChooser, err := initProxyChooser(app.config.proxyFilePath)
	if err != nil {
		panic(err)
	} else if proxyChooser != nil {
		options = append(options, crawler.WithProxyChooser(proxyChooser))
	}
	userAgentChooser, err := initUserAgentChooser(app.config.agentsFilePath)
	if err != nil {
		panic(err)
	} else if userAgentChooser != nil {
		options = append(options, crawler.WithUserAgentChooser(userAgentChooser))
	}
	filestore := store.NewFileStore(os.Getenv("FILESTORE_OUT_DIR"))
	app.crawler = *crawler.NewCrawler(&app.cache, filestore, options...)

	app.seed(ctx)
	app.crawl(ctx)
}
