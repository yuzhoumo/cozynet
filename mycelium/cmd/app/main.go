package main

import (
	"context"
	"mycelium/internal/cache"
	"mycelium/internal/crawler"
	"mycelium/internal/filter"
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
	redisCacheOptions := cache.CrawlerCacheOptions{
		Addr: env.RedisAddr,
		Pass: env.RedisPass,
		DB:   env.RedisDB,
	}
	if cache, err := cache.NewRedisCache(ctx, &redisCacheOptions); err != nil {
		panic(err)
	} else {
		app.cache = *cache
	}

	// create crawler options
	options := []crawler.CrawlerOption{}
	options = append(options, crawler.WithMaxIdle(app.config.maxIdleSeconds))
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
	if domainBlacklist, err := initDomainBlacklist(app.config.domainBlacklistFile); err != nil {
		panic(err)
	} else if domainBlacklist != nil {
		filter := filter.NewDomainFilter(domainBlacklist)
		options = append(options, crawler.WithUrlFilters([]crawler.UrlFilter{filter}))
	}

	// Add fungicide integration options
	if env.FungicideQueueKey != "" {
		options = append(options, crawler.WithFungicideQueueKey(env.FungicideQueueKey))
	}
	if env.MyceliumIngressKey != "" {
		options = append(options, crawler.WithMyceliumIngressKey(env.MyceliumIngressKey))
	}
	if env.MyceliumBlacklistKey != "" {
		options = append(options, crawler.WithMyceliumBlacklistKey(env.MyceliumBlacklistKey))
	}

	filestore := store.NewFileStore(env.FilestoreOutDir)
	app.crawler = *crawler.NewCrawler(&app.cache, filestore, options...)

	app.seed(ctx)

	// Run crawler and ingress consumer concurrently if fungicide integration is enabled
	if env.MyceliumIngressKey != "" {
		go app.consumeIngress(ctx)
	}

	app.crawl(ctx)
}
