package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"mycelium/internal/chooser"

	"github.com/mroth/weightedrand/v2"
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

func loadUserAgentOptions(path string) ([]chooser.UserAgentOption, error) {
	var options []chooser.UserAgentOption

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", path, err)
	}

	err = json.Unmarshal(content, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", path, err)
	}

	return options, nil
}

func initUserAgentChooser(path string) (*chooser.UserAgentChooser, error) {
	userAgentOptions, err := loadUserAgentOptions(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent file %s: %w", path, err)
	}

	var choices []weightedrand.Choice[string, int]
	for _, opt := range userAgentOptions {
		choices = append(choices, weightedrand.NewChoice(opt.UserAgent, opt.Percent))
	}

	randomChooser, err := weightedrand.NewChooser(choices...)
	if err != nil {
		return nil, fmt.Errorf("failed to create weighted random chooser: %w", err)
	}

	return chooser.NewUserAgentChooser(userAgentOptions, randomChooser), nil
}
