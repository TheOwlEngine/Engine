package lib

import (
	"encoding/json"
	"engine/types"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gosimple/slug"
	"github.com/icza/mjpeg"
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

func RenderVideo(name string, pageId string, videoDirectory string, globalErrors []string) (string, string, []string) {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	log.Printf("%s Render recorded video", yellow("[ Engine ]"))

	slugName := slug.Make(name)
	videoName := slugName + "-" + pageId + ".mp4"
	videoPath := videoDirectory + videoName

	go func() {
		renderer, errorMjpeg := mjpeg.New(videoPath, int32(1440), int32(900), 6)

		if errorMjpeg != nil {
			log.Printf(red("[ Engine ] %v\n"), errorMjpeg)
			globalErrors = append(globalErrors, `Failed to create temporary motion image`)
		}

		matches, errorGlobFile := filepath.Glob(videoDirectory + pageId + "-*-frame.jpeg")

		if errorGlobFile != nil {
			log.Printf(red("[ Engine ] %v\n"), errorGlobFile)
			globalErrors = append(globalErrors, `Failed to list generated motion image`)
		}

		sort.Strings(matches)

		for _, name := range matches {
			data, errorReadFile := ioutil.ReadFile(name)

			if errorReadFile != nil {
				log.Printf(red("[ Engine ] %v\n"), errorReadFile)
				globalErrors = append(globalErrors, `Failed to read rendered motion image`)
			}

			renderer.AddFrame(data)
		}

		renderer.Close()

		compressedPath := strings.ReplaceAll(videoPath, ".mp4", "-compressed.mp4")

		cmd := exec.Command("ffmpeg", "-i", videoPath, "-vcodec", "h264", "-acodec", "mp2", compressedPath)
		stdout, errorFFmpeg := cmd.Output()

		if errorFFmpeg != nil {
			log.Printf(red("[ Engine ] %v\n"), errorFFmpeg)
			globalErrors = append(globalErrors, `Failed to compress temporary motion image`)
		}

		if len(stdout) > 0 {
			log.Printf("%s %v\n", yellow("[ Engine ]"), stdout)
		}

		time.Sleep(2 * time.Second)

		for _, name := range matches {
			errorRemoveFile := os.Remove(name)

			if errorRemoveFile != nil {
				log.Printf(red("[ Engine ] %v\n"), errorRemoveFile)
				globalErrors = append(globalErrors, `Failed to remove rendered motion image`)
			}
		}

		errorRemoveTemporary := os.Remove(videoPath)

		if errorRemoveTemporary != nil {
			log.Printf(red("[ Engine ] %v\n"), errorRemoveTemporary)
			globalErrors = append(globalErrors, `Failed to remove temporary motion image`)
		}

		errorRemoveCompressed := os.Rename(compressedPath, videoPath)

		if errorRemoveCompressed != nil {
			log.Printf(red("[ Engine ] %v\n"), errorRemoveCompressed)
			globalErrors = append(globalErrors, `Failed to remove compressed motion image`)
		}
	}()

	return videoName, videoPath, globalErrors
}
