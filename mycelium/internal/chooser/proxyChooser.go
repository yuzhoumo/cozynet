package chooser

import "fmt"

type ProxyOption struct {
	Username string `json:"user"`
	Password string `json:"pass"`
	URL      string `json:"url"`
}

func (po *ProxyOption) String() string {
	return fmt.Sprintf("%s", "") //TODO: implement
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
	// TODO: implement
	return nil, nil
}

func (pc *ProxyChooser) Pick() string {
	choice := pc.options[pc.index]
	pc.index = (pc.index + 1) % len(pc.options)
	return choice.String()
}
