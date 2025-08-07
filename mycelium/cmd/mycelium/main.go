package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"mycelium/internal/crawler"
)

func main() {
    var seedFilePath string

    flag.StringVar(&seedFilePath, "seedfile", "./seed.txt", "newline delimited list of seed urls")
    flag.Parse()

    urls, err := loadSeedUrls(seedFilePath)
    if err != nil {
        panic(err)
    }

    fmt.Printf("successfully parsed %d urls\n", len(urls))

    ctx := context.Background()
    req := crawler.New(&http.Client{})

    for i := 0; i < 10; i++ {
        fmt.Printf("Requesting %s\n", urls[i].String())
        req.GetPageContent(ctx, urls[i])
    }
}
