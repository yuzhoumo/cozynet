package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"mycelium/internal/chooser"
)

func initCliFlags(conf *MyceliumConfig) {
	flag.StringVar(&conf.seedFile, "seedfile", "", "newline delimited list of seed urls")
	flag.StringVar(&conf.agentsFile, "agentsfile", "", "user agents json")
	flag.StringVar(&conf.proxyFile, "proxyfile", "", "proxy list json")
	flag.StringVar(&conf.domainBlacklistFile, "domainsblacklist", "", "newline delimited list of blacklisted domains")
	flag.IntVar(&conf.numCrawlers, "routines", 1, "number of crawler routines to spawn")
	flag.IntVar(&conf.maxIdleSeconds, "maxIdleSeconds", 100, "max seconds to wait for queue items before crawler exits")
	flag.Parse()
}

func initEnvironment(env *Environment) error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	redisDB, err := strconv.ParseInt(os.Getenv("REDIS_DB"), 10, 0)
	if err != nil {
		return err
	}

	env.RedisAddr = os.Getenv("REDIS_ADDR")
	env.RedisDB = int(redisDB)
	env.RedisPass = os.Getenv("REDIS_PASS")
	env.FilestoreOutDir = os.Getenv("FILESTORE_OUT_DIR")

	return nil
}

func initDomainBlacklist(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	domainfile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open blacklist file %s: %w", path, err)
	}
	defer domainfile.Close()

	var res []string
	scanner := bufio.NewScanner(domainfile)
	line := 1

	for scanner.Scan() {
		domain := scanner.Text()
		res = append(res, domain)
		line++
	}

	return res, nil
}

func initSeedUrls(path string) ([]*url.URL, error) {
	seedfile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open seed file %s: %w", path, err)
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
