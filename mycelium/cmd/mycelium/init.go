package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"mycelium/internal/chooser"
)

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

func initUserAgentChooser(path string) (*chooser.UserAgentChooser, error) {
	userAgentOptions, err := chooser.LoadUserAgentOptions(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent file %s: %w", path, err)
	}
	return chooser.NewUserAgentChooser(userAgentOptions)
}
