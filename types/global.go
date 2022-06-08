package types

import "time"

type Result struct {
	Id             string        `json:"id,omitempty"`
	Code           int           `json:"code"`
	Name           string        `json:"name,omitempty"`
	Slug           string        `json:"slug,omitempty"`
	Proxy          string        `json:"proxy,omitempty"`
	Message        string        `json:"message,omitempty"`
	Duration       time.Duration `json:"duration,omitempty"`
	Engine         string        `json:"engine,omitempty"`
	FirstPage      string        `json:"first_page,omitempty"`
	ItemsOnPage    int           `json:"items_on_page"`
	Infinite       bool          `json:"infinite"`
	InfiniteScroll int           `json:"infinite_scroll"`
	Paginate       bool          `json:"paginate"`
	PaginateLimit  int           `json:"paginate_limit"`
	Record         bool          `json:"record"`
	Recording      string        `json:"recording,omitempty"`
	Result         []ResultPage  `json:"result,omitempty"`
	Usage          ResultUsage   `json:"usage,omitempty"`
	Errors         []string      `json:"errors,omitempty"`
}

type ResultPage struct {
	Title    string          `json:"title,omitempty"`
	Url      string          `json:"url,omitempty"`
	Page     int             `json:"page"`
	Duration time.Duration   `json:"duration,omitempty"`
	Content  []ResultContent `json:"content,omitempty"`
}

type ResultContent struct {
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
	InfiniteScroll int    `yaml:"infinite_scroll"`
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
	Scroll         int     `yaml:"scroll"`
	Capture        Capture `yaml:"capture"`
	Table          Table   `yaml:"table"`
	Wrapper        string  `yaml:"wrapper"`
}

type Element struct {
	Selector string   `yaml:"selector"`
	Contains Contains `yaml:"contains"`
	Write    string   `yaml:"write"`
	Value    string   `yaml:"value"`
	Select   string   `yaml:"select"`
	Multiple []string `yaml:"multiple"`
	Check    string   `yaml:"check"`
	Radio    string   `yaml:"radio"`
	// Upload   string   `yaml:"upload"`
	Action string `yaml:"action"`
}

type Take struct {
	Name           string   `yaml:"name"`
	Selector       string   `yaml:"selector"`
	Contains       Contains `yaml:"contains"`
	NextToSelector string   `yaml:"next_to_selector"`
	NextToContains Contains `yaml:"next_to_contains"`
	Parse          string   `yaml:"parse"`
	UseForNavigate bool     `yaml:"use_for_navigate"`
}

type Contains struct {
	Selector   string `yaml:"selector"`
	Identifier string `yaml:"identifier"`
}

type Capture struct {
	Selector string      `yaml:"selector"`
	Name     string      `yaml:"name"`
	Delay    int         `yaml:"delay"`
	Clip     CaptureClip `yaml:"clip"`
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
