package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"mycelium/internal/chooser"
)

func initCliFlags(conf *MyceliumConfig) {
	flag.StringVar(&conf.seedFilePath, "seedfile", "", "newline delimited list of seed urls")
	flag.StringVar(&conf.agentsFilePath, "agentsfile", "", "user agents json")
	flag.StringVar(&conf.proxyFilePath, "proxyfile", "", "proxy list json")
	flag.IntVar(&conf.crawlRoutines, "routines", 1, "number of crawler routines to spawn")
	flag.Parse()
}

func initSeedUrls(path string) ([]*url.URL, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to seed file %s: %w", path, err)
	}

	seedfile, err := os.Open(abspath)
	if err != nil {
		return nil, fmt.Errorf("failed to open seed file %s: %w", abspath, err)
	}
	defer seedfile.Close()

	var res []*url.URL
	scanner := bufio.NewScanner(seedfile)
	line := 1

	for scanner.Scan() {
		rawUrl := scanner.Text()
		url, err := url.Parse(rawUrl)

		if err != nil {
			return nil, fmt.Errorf("failed to parse seed file line %d: %s", line, rawUrl)
		}

		res = append(res, url)
		line++
	}

	return res, nil
}

func initProxyChooser(path string) (*chooser.ProxyChooser, error) {
	if path == "" {
		return nil, nil
	}
	options, err := chooser.LoadProxyOptions(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load proxy file %s: %w", path, err)
	}
	return chooser.NewProxyChooser(options), nil
}

func initUserAgentChooser(path string) (*chooser.UserAgentChooser, error) {
	if path == "" {
		return nil, nil
	}
	userAgentOptions, err := chooser.LoadUserAgentOptions(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent file %s: %w", path, err)
	}
	return chooser.NewUserAgentChooser(userAgentOptions)
}
