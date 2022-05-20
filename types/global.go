package types

type Clip struct {
	Top    float64 `yaml:"top"`
	Left   float64 `yaml:"left"`
	Width  float64 `yaml:"width"`
	Height float64 `yaml:"height"`
}

type Screenshot struct {
	Path    string  `yaml:"path"`
	Timeout float64 `yaml:"timeout"`
	Clip    Clip    `yaml:"clip"`
}

type Element struct {
	Selector       string `yaml:"selector"`
	Contains       string `yaml:"contains"`
	NextToSelector string `yaml:"next_to_selector"`
	NextToContains string `yaml:"next_to_contains"`
	Name           string `yaml:"name"`
	Result         string `yaml:"result"`
}

type Step struct {
	Id         string
	Action     string     `yaml:"action"`
	Delay      float64    `yaml:"delay"`
	Element    string     `yaml:"element"`
	Take       []Element  `yaml:"extract"`
	Screenshot Screenshot `yaml:"screenshot"`
	Write      string     `yaml:"write"`
}

type Flow struct {
	Name   string `yaml:"name"`
	Repeat int    `yaml:"repeat"`
	Step   []Step `yaml:"step"`
}

type Config struct {
	Engine  string `yaml:"engine"`
	WebPage string `yaml:"webpage"`
	Flow    []Flow `yaml:"flow"`
}

type Request struct {
	Engine   string
	Flow     []Flow
	Height   float64
	HtmlOnly string
	Path     string
	Timeout  float64
	Type     string
	WebPage  string
	Width    float64
}

type Response struct {
	Id      string            `json:"id,omitempty"`
	Code    int               `json:"code,omitempty"`
	Message string            `json:"message,omitempty"`
	Html    map[string]string `json:"html,omitempty"`
	Path    string            `json:"path,omitempty"`
}
