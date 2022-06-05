package types

import "time"

type Result struct {
	Id             string        `json:"id,omitempty"`
	Code           int           `json:"code"`
	Name           string        `json:"name,omitempty"`
	Slug           string        `json:"slug,omitempty"`
	Message        string        `json:"message,omitempty"`
	Duration       time.Duration `json:"duration,omitempty"`
	Engine         string        `json:"engine,omitempty"`
	FirstPage      string        `json:"first_page,omitempty"`
	ItemsOnPage    int           `json:"items_on_page"`
	Infinite       bool          `json:"infinite"`
	InfiniteDelay  bool          `json:"infinite_delay"`
	Paginate       bool          `json:"paginate"`
	PaginateButton string        `json:"paginate_button,omitempty"`
	PaginateLimit  int           `json:"paginate_limit"`
	Record         bool          `json:"record"`
	Recording      string        `json:"recording,omitempty"`
	Result         []ResultPage  `json:"result,omitempty"`
	Usage          ResultUsage   `json:"usage,omitempty"`
}

type ResultPage struct {
	Page     int             `json:"page"`
	Duration time.Duration   `json:"duration,omitempty"`
	Content  []ResultContent `json:"content,omitempty"`
}

type ResultContent struct {
	Title   string `json:"title,omitempty"`
	Url     string `json:"url,omitempty"`
	Page    int    `json:"page"`
	Type    string `json:"type,omitempty"`
	Length  int    `json:"length"`
	Name    string `json:"name,omitempty"`
	Content string `json:"content,omitempty"`
}

type ResultUsage struct {
	Disk      map[string]float64 `json:"disk,omitempty"`
	Bandwidth map[string]float64 `json:"bandwidth,omitempty"`
}

type Config struct {
	Name           string `yaml:"name"`
	Engine         string `yaml:"engine"`
	FirstPage      string `yaml:"first_page"`
	ItemsOnPage    int    `yaml:"items_on_page"`
	Infinite       bool   `yaml:"infinite"`
	InfiniteDelay  bool   `yaml:"infinite_delay"`
	Paginate       bool   `yaml:"paginate"`
	PaginateButton string `yaml:"paginate_button"`
	PaginateLimit  int    `yaml:"paginate_limit"`
	Proxy          bool   `yaml:"proxy"`
	ProxyCountry   string `yaml:"proxy_country"`
	Record         bool   `yaml:"record"`
	Flow           []Flow `yaml:"flow"`
}

type Flow struct {
	Element        Element `yaml:"element"`
	Take           Take    `yaml:"take"`
	Navigate       bool    `yaml:"navigate"`
	BackToPrevious bool    `yaml:"back_to_previous"`
	WaitFor        string  `yaml:"wait_for"`
	Delay          int     `yaml:"delay"`
	Capture        Capture `yaml:"capture"`
	Table          Table   `yaml:"table"`
	Wrapper        string  `yaml:"wrapper"`
}

type Element struct {
	Selector string `yaml:"selector"`
	Write    string `yaml:"write"`
	Value    string `yaml:"value"`
	Choose   string `yaml:"choose"`
	Upload   string `yaml:"upload"`
	Action   string `yaml:"action"`
	Key      string `yaml:"key"`
}

type Take struct {
	Name           string       `yaml:"name"`
	Selector       string       `yaml:"selector"`
	Contains       TakeContains `yaml:"contains"`
	NextToSelector string       `yaml:"next_to_selector"`
	NextToContains TakeContains `yaml:"next_to_contains"`
	Parse          string       `yaml:"parse"`
	UseForNavigate bool         `yaml:"use_for_navigate"`
}

type TakeContains struct {
	Selector   string `yaml:"selector"`
	Identifier string `yaml:"identifier"`
}

type Capture struct {
	Path  string      `yaml:"path"`
	Delay int         `yaml:"delay"`
	Clip  CaptureClip `yaml:"clip"`
}

type CaptureClip struct {
	Top    float64 `yaml:"top"`
	Left   float64 `yaml:"left"`
	Width  float64 `yaml:"width"`
	Height float64 `yaml:"height"`
}

type Table struct {
	Selector string       `yaml:"selector"`
	Name     string       `yaml:"name"`
	Fields   []TableField `yaml:"fields"`
}

type TableField struct {
	Index int    `yaml:"index"`
	Name  string `yaml:"name"`
}
