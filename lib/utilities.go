package lib

import (
	"encoding/json"
	"engine/types"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

var replacerQuote = strings.NewReplacer(`"`, `$"`, "\n", "$\n")
var replacerJson = strings.NewReplacer(`"{`, `{`, `}"`, `}`, `"[`, `[`, `]"`, `]`, `$\"`, `"`, `$\n`, "\n", `\!`, `!`, `\@`, `@`, `\#`, `#`, `\$`, `$`, `\%`, `%`, `\^`, `^`, `\&`, `&`, `\*`, `*`, `\(`, `(`, `\)`, `)`, `\-`, `-`, `\+`, `+`, `\_`, `_`)

func Response(w http.ResponseWriter, data types.Result, pageId string) {
	yellow := color.New(color.FgYellow).SprintFunc()

	if pageId != "" {
		log.Printf("%s Sending result", yellow("[ Engine ]"))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	for _, page := range data.Result {
		for index := range page.Content {
			if strings.Contains(page.Content[index].Content, `[`) {
				page.Content[index].Content = replacerQuote.Replace(page.Content[index].Content)
			}
		}
	}

	jsonTable, _ := json.Marshal(data)
	jsonEncoded := Unescape(jsonTable)
	jsonResult := replacerJson.Replace(jsonEncoded)

	w.Write([]byte(jsonResult))
}

func Noop(w http.ResponseWriter, r *http.Request) {}

func Int(integer int) *int {
	return &integer
}

func Unescape(json json.RawMessage) string {
	result, errorUnquote := strconv.Unquote(strings.Replace(strconv.Quote(string(json)), `\\u`, `\u`, -1))

	if errorUnquote != nil {
		return ""
	}

	return result
}

func Contains(sl []string, name string) bool {
	for _, value := range sl {
		if value == name {
			return true
		}
	}

	return false
}

func Cors(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}
