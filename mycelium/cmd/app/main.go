package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"mycelium/internal/crawler"
	"mycelium/internal/redisq"

	"github.com/google/uuid"
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

	redisDb, err := strconv.ParseInt(os.Getenv("REDIS_DB"), 10, 0)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	queue := redisq.NewRedisQueue(&redisq.RedisQueueOptions{
		Addr: os.Getenv("REDIS_ADDR"),
		Pass: os.Getenv("REDIS_PASS"),
		DB:   int(redisDb),
	})
	crawl := crawler.NewCrawler(queue, options...)

	for i := 0; i < 10; i++ {
		fmt.Printf("Requesting %s\n", urls[i].String())
		page, err := crawl.GetPage(ctx, urls[i])
		if err != nil {
			panic(err)
		}

		data, err := page.MarshalJson()
		if err != nil {
			panic(err)
		}

		if err := os.WriteFile("./out/"+uuid.New().String()+".json", data, 0644); err != nil {
			panic(err)
		}

	}
}
