package main

import (
	"context"
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
	redisCacheOptions := redis.RedisCacheOptions{
		Addr: env.RedisAddr,
		Pass: env.RedisPass,
		DB:   env.RedisDB,
	}
	if cache, err := redis.NewRedisCache(ctx, &redisCacheOptions); err != nil {
		panic(err)
	} else {
		app.cache = *cache
	}

	// create crawler options
	options := []crawler.CrawlerOption{}
	if proxyChooser, err := initProxyChooser(app.config.proxyFile); err != nil {
		panic(err)
	} else if proxyChooser != nil {
		options = append(options, crawler.WithProxyChooser(proxyChooser))
	}
	if uaChooser, err := initUserAgentChooser(app.config.agentsFile); err != nil {
		panic(err)
	} else if uaChooser != nil {
		options = append(options, crawler.WithUserAgentChooser(uaChooser))
	}

	filestore := store.NewFileStore(env.FilestoreOutDir)
	app.crawler = *crawler.NewCrawler(&app.cache, filestore, options...)

	app.seed(ctx)
	app.crawl(ctx)
}
