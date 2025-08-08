package chooser

type StringChooser interface {
	Pick() string
}

type UserAgentOption struct {
	UserAgent string `json:"ua"`
	Percent   int    `json:"pct"`
}

type UserAgentChooser struct {
	options []UserAgentOption
	chooser StringChooser
}

func NewUserAgentChooser(options []UserAgentOption, chooser StringChooser) *UserAgentChooser {
	return &UserAgentChooser{
		options: options,
		chooser: chooser,
	}
}

func (uac *UserAgentChooser) Choose() string {
	return uac.chooser.Pick()
}
