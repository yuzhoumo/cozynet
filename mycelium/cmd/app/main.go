package main

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"sync"

	"mycelium/internal/crawler"
	"mycelium/internal/redis"
	"mycelium/internal/store"

	"github.com/joho/godotenv"
)

type MyceliumConfig struct {
	seedFilePath   string
	agentsFilePath string
	proxyFilePath  string
	crawlRoutines  int
}

type Mycelium struct {
	config MyceliumConfig
}

func main() {
	var app Mycelium

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

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

	redisQueueDb, err := strconv.ParseInt(os.Getenv("REDIS_QUEUE_DB"), 10, 0)
	if err != nil {
		panic(err)
	}

	queue := redis.NewRedisQueue(&redis.RedisQueueOptions{
		Addr: os.Getenv("REDIS_ADDR"),
		Pass: os.Getenv("REDIS_PASS"),
		DB:   int(redisQueueDb),
	})

	redisVisitedDb, err := strconv.ParseInt(os.Getenv("REDIS_VISITED_DB"), 10, 0)
	if err != nil {
		panic(err)
	}

	visited := redis.NewRedisVisited(&redis.RedisVisitedOptions{
		Addr: os.Getenv("REDIS_ADDR"),
		Pass: os.Getenv("REDIS_PASS"),
		DB:   int(redisVisitedDb),
	})

	var seed []crawler.QueueItem
	for _, seedUrl := range urls {
		seed = append(seed, redis.NewQueueItem(seedUrl))
	}

	filestore := store.NewFileStore(os.Getenv("FILESTORE_OUT_DIR"))

	ctx := context.Background()
	crawl := crawler.NewCrawler(queue, visited, filestore, options...)
	err = crawl.Seed(ctx, seed)
	if err != nil {
		panic(err)
	}

	makeQueueItem := func(u *url.URL) crawler.QueueItem {
		return redis.NewQueueItem(u)
	}

	crawlRoutine := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = crawl.Crawl(ctx, makeQueueItem)
		if err != nil {
			panic(err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(conf.crawlRoutines)

	for i := 0; i < conf.crawlRoutines; i++ {
		go crawlRoutine(&wg)
	}

	wg.Wait()
}
