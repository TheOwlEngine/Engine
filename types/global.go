package types

type Request struct {
	Engine   string
	Flow     []Flow
	Height   float64
	HtmlOnly string
	Path     string
	Timeout  float64
	Type     string
	Target   string
	Width    float64
}

type Response struct {
	Id      string            `json:"id,omitempty"`
	Code    int               `json:"code,omitempty"`
	Message string            `json:"message,omitempty"`
	Html    map[string]string `json:"html,omitempty"`
	Path    string            `json:"path,omitempty"`
}

type Config struct {
	Engine string `yaml:"engine"`
	Target string `yaml:"target"`
	Name   string `yaml:"name"`
	Repeat int    `yaml:"repeat"`
	Flow   []Flow `yaml:"flow"`
}

type Flow struct {
	Take       []Element  `yaml:"take"`
	Selector   Selector   `yaml:"selector"`
	Navigate   string     `yaml:"navigate"`
	Delay      float64    `yaml:"delay"`
	Screenshot Screenshot `yaml:"screenshot"`
}

type Selector struct {
	Selector string `yaml:"selector"`
	Fill     string `yaml:"fill"`
	Do       string `yaml:"do"`
}

type Element struct {
	Selector       string          `yaml:"selector"`
	Contains       ElementContains `yaml:"contains"`
	NextToSelector string          `yaml:"next_to_selector"`
	NextToContains ElementContains `yaml:"next_to_contains"`
	Name           string          `yaml:"name"`
	Parse          string          `yaml:"parse"`
}

type ElementContains struct {
	Selector string `yaml:"selector"`
	Text     string `yaml:"text"`
}

type Screenshot struct {
	Path    string  `yaml:"path"`
	Timeout float64 `yaml:"timeout"`
	Clip    Clip    `yaml:"clip"`
}

type Clip struct {
	Top    float64 `yaml:"top"`
	Left   float64 `yaml:"left"`
	Width  float64 `yaml:"width"`
	Height float64 `yaml:"height"`
}
