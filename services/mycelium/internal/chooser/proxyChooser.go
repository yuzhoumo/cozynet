package chooser

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
)

type ProxyOption struct {
	URL url.URL
}

func (po *ProxyOption) String() string {
	return po.URL.String()
}

type ProxyChooser struct {
	options []ProxyOption
	index   int
}

func NewProxyChooser(options []ProxyOption) *ProxyChooser {
	return &ProxyChooser{
		options: options,
		index:   0,
	}
}

func LoadProxyOptions(path string) ([]ProxyOption, error) {
	proxyfile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open proxy file %s: %w", path, err)
	}
	defer proxyfile.Close()

	var options []ProxyOption
	scanner := bufio.NewScanner(proxyfile)
	line := 1

	for scanner.Scan() {
		rawUrl := scanner.Text()
		parsedUrl, err := url.Parse(rawUrl)

		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy file line %d: %s", line, rawUrl)
		}

		options = append(options, ProxyOption{URL: *parsedUrl})
		line++
	}

	return options, nil
}

func (pc *ProxyChooser) Pick() string {
	choice := pc.options[pc.index]
	pc.index = (pc.index + 1) % len(pc.options)
	fmt.Println(choice.String())
	return choice.String()
}
