package main

import (
	"context"
	"net/url"
	"os"
	"strconv"

	"mycelium/internal/crawler"
	"mycelium/internal/redis"

	"github.com/joho/godotenv"
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

	ctx := context.Background()
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

	crawl := crawler.NewCrawler(queue, visited, options...)
	crawl.Crawl(ctx, seed, func(u *url.URL) crawler.QueueItem { return redis.NewQueueItem(u) })

	// fmt.Printf("Requesting %s\n", urls[i].String())
	// page, err := crawl.GetPage(ctx, urls[i])
	// if err != nil {
	//     panic(err)
	// }

	// data, err := page.MarshalJson()
	// if err != nil {
	//     panic(err)
	// }

	// if err := os.WriteFile("./out/"+uuid.New().String()+".json", data, 0644); err != nil {
	//     panic(err)
	// }
}
