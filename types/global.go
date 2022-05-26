package types

type Response struct {
	Id         string                    `json:"id,omitempty"`
	Code       int                       `json:"code,omitempty"`
	Name       string                    `json:"name,omitempty"`
	Target     string                    `json:"target,omitempty"`
	Engine     string                    `json:"engine,omitempty"`
	Message    string                    `json:"message,omitempty"`
	Result     map[int]map[string]string `json:"html,omitempty"`
	Screenshot map[string]string         `json:"screenshot,omitempty"`
	Video      map[string]string         `json:"video,omitempty"`
}

type Config struct {
	Name     string `yaml:"name"`
	Engine   string `yaml:"engine"`
	Flow     []Flow `yaml:"flow"`
	HtmlOnly bool   `yaml:"html_only"`
	Paginate bool   `yaml:"paginate"`
	Repeat   int    `yaml:"repeat"`
	Target   string `yaml:"target"`
	Record   bool   `yaml:"record"`
}

type Flow struct {
	Take       []Element  `yaml:"take"`
	Form       Form       `yaml:"form"`
	Navigate   string     `yaml:"navigate"`
	Delay      float64    `yaml:"delay"`
	Screenshot Screenshot `yaml:"screenshot"`
}

type Form struct {
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
	Table          ElementTable    `yaml:"table"`
}

type ElementContains struct {
	Selector   string `yaml:"selector"`
	Identifier string `yaml:"identifier"`
}

type ElementTable struct {
	Selector string              `yaml:"selector"`
	Name     string              `yaml:"name"`
	WithCSV  bool                `yaml:"with_csv"`
	Fields   []ElementTableField `yaml:"fields"`
}

type ElementTableField struct {
	Index int    `yaml:"index"`
	Name  string `yaml:"name"`
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
