package chooser

import (
    "encoding/json"
    "fmt"
    "os"

	"github.com/mroth/weightedrand/v2"
)

type UserAgentOption struct {
	UserAgent string `json:"ua"`
	Percent   int    `json:"pct"`
}

func (uao *UserAgentOption) String() string {
    return uao.UserAgent
}

type UserAgentChooser struct {
	weightedRandomChooser *weightedrand.Chooser[string, int]
}

func NewUserAgentChooser(options []UserAgentOption) (*UserAgentChooser, error) {
	var choices []weightedrand.Choice[string, int]
	for _, opt := range options {
		choices = append(choices, weightedrand.NewChoice(opt.UserAgent, opt.Percent))
	}

    chooser, err := weightedrand.NewChooser(choices...)
    if err != nil {
        return nil, err
    }

	return &UserAgentChooser{ weightedRandomChooser: chooser }, nil
}

func LoadUserAgentOptions(path string) ([]UserAgentOption, error) {
	var options []UserAgentOption

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

func (uac *UserAgentChooser) Pick() string {
	return uac.weightedRandomChooser.Pick()
}
