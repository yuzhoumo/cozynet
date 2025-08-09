package chooser

import (
    "encoding/json"
    "fmt"
    "os"
)

type ProxyOption struct {
	Username string `json:"user"`
	Password string `json:"pass"`
	URL      string `json:"url"`
}

func (po *ProxyOption) String() string {
    if po.Username != "" && po.Password != "" {
        return fmt.Sprintf("http://%s:%s@%s", po.Username, po.Password, po.URL)
    }
	return fmt.Sprintf("http://%s", po.URL)
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
	var options []ProxyOption

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

func (pc *ProxyChooser) Pick() string {
	choice := pc.options[pc.index]
	pc.index = (pc.index + 1) % len(pc.options)
	return choice.String()
}
