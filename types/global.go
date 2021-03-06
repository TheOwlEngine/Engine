package types

import "time"

type Proxy struct {
	IP        string    `json:"ip"`
	UserAgent UserAgent `json:"user_agent"`
}

type UserAgent struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	OS        string `json:"os"`
	OSVersion string `json:"osversion"`
	Device    string `json:"device"`
	Mobile    bool   `json:"mobile"`
	Tablet    bool   `json:"tablet"`
	Desktop   bool   `json:"desktop"`
	Bot       bool   `json:"bot"`
	URL       string `json:"url"`
	String    string `json:"string"`
}

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

type ResultTable struct {
	Name   string              `json:"name"`
	Column int                 `json:"column"`
	Row    int                 `json:"row"`
	Header []ResultTableHead   `json:"header"`
	Data   [][]ResultTableData `json:"data"`
}

type ResultTableHead struct {
	Index   int    `json:"index"`
	Length  int    `json:"length"`
	Content string `json:"content,omitempty"`
}

type ResultTableData struct {
	Type    string `json:"type,omitempty"`
	Index   int    `json:"index"`
	Length  int    `json:"length"`
	Name    string `json:"name,omitempty"`
	Content string `json:"content,omitempty"`
}

type Config struct {
	Name           string `yaml:"name" json:"name"`
	Engine         string `yaml:"engine" json:"engine"`
	FirstPage      string `yaml:"first_page" json:"first_page"`
	ItemsOnPage    int    `yaml:"items_on_page" json:"items_on_page"`
	Infinite       bool   `yaml:"infinite" json:"infinite"`
	InfiniteScroll int    `yaml:"infinite_scroll" json:"infinite_scroll"`
	Paginate       bool   `yaml:"paginate" json:"paginate"`
	PaginateButton string `yaml:"paginate_button" json:"paginate_button"`
	PaginateLimit  int    `yaml:"paginate_limit" json:"paginate_limit"`
	Proxy          bool   `yaml:"proxy" json:"proxy"`
	ProxyCountry   string `yaml:"proxy_country" json:"proxy_country"`
	Record         bool   `yaml:"record" json:"record"`
	Flow           []Flow `yaml:"flow" json:"flow"`
}

type Flow struct {
	Element        Element `yaml:"element" json:"element"`
	Take           Take    `yaml:"take" json:"take"`
	Navigate       bool    `yaml:"navigate" json:"navigate"`
	BackToPrevious bool    `yaml:"back_to_previous" json:"back_to_previous"`
	WaitFor        WaitFor `yaml:"wait_for" json:"wait_for"`
	Delay          int     `yaml:"delay" json:"delay"`
	Scroll         int     `yaml:"scroll" json:"scroll"`
	Wrapper        string  `yaml:"wrapper" json:"wrapper"`
	Capture        Capture `yaml:"capture" json:"capture"`
	Table          Table   `yaml:"table" json:"table"`
}

type Element struct {
	Selector string   `yaml:"selector" json:"selector"`
	Contains Contains `yaml:"contains" json:"contains"`
	Write    string   `yaml:"write" json:"write"`
	Value    string   `yaml:"value" json:"value"`
	Select   string   `yaml:"select" json:"select"`
	Multiple []string `yaml:"multiple" json:"multiple"`
	Action   string   `yaml:"action" json:"action"`
}

type Take struct {
	Name           string   `yaml:"name" json:"name"`
	Selector       string   `yaml:"selector" json:"selector"`
	Contains       Contains `yaml:"contains" json:"contains"`
	Parse          string   `yaml:"parse" json:"parse"`
	UseForNavigate bool     `yaml:"use_for_navigate" json:"use_for_navigate"`
}

type Contains struct {
	Selector   string `yaml:"selector" json:"selector"`
	Identifier string `yaml:"identifier" json:"identifier"`
}

type WaitFor struct {
	Selector string `yaml:"selector" json:"selector"`
	Delay    int    `yaml:"delay" json:"delay"`
}

type Capture struct {
	Selector string      `yaml:"selector" json:"selector"`
	Name     string      `yaml:"name" json:"name"`
	Delay    int         `yaml:"delay" json:"delay"`
	Clip     CaptureClip `yaml:"clip" json:"clip"`
}

type CaptureClip struct {
	Top    float64 `yaml:"top" json:"top"`
	Left   float64 `yaml:"left" json:"left"`
	Width  float64 `yaml:"width" json:"width"`
	Height float64 `yaml:"height" json:"height"`
}

type Table struct {
	Selector string   `yaml:"selector" json:"selector"`
	Name     string   `yaml:"name" json:"name"`
	Fields   []string `yaml:"fields" json:"fields"`
}
