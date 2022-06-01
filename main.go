package main

import (
	"context"
	"encoding/json"
	"engine/types"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/fatih/color"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/icza/mjpeg"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
	"github.com/ysmood/gson"
	"github.com/yukithm/json2csv"
	"github.com/yukithm/json2csv/jsonpointer"
	"golang.org/x/net/html"
)

// var engineSession string
var engineBrowser rod.Browser

var enginePort string
var engineProxy string
var engineDebug bool

var useProxy bool = false

var rootDirectory string
var resourcesDirectory string
var screenshotDirectory string
var videoDirectory string
var logsDirectory string

var defaultTimeout time.Duration

// TODO Comment
// ....
func main() {
	godotenv.Load(".env")

	defaultTimeout = 3 * time.Second

	rootDirectory, _ = os.Getwd()

	resourcesDirectory = rootDirectory + "/resources/"
	screenshotDirectory = resourcesDirectory + "/screenshots/"
	videoDirectory = resourcesDirectory + "/videos/"
	logsDirectory = rootDirectory + "/logs/"

	// log to custom file
	currentTime := time.Now()
	currentDate := fmt.Sprintf("%d-%02d-%02d", currentTime.Year(), currentTime.Month(), currentTime.Day())
	rotatingLogFile := logsDirectory + currentDate + ".log"

	// open log file
	logFile, logError := os.OpenFile(rotatingLogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	multiLogger := io.MultiWriter(logFile, os.Stdout)

	if logError != nil {
		log.Panic(logError)
		panic(logError)
	}

	defer logFile.Close()

	log.SetOutput(multiLogger)
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	yellow := color.New(color.FgYellow).SprintFunc()

	app := &cli.App{
		Name:  "Engine",
		Usage: "This application will provide a browser base engine of the web scraper",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "port",
				Value: "3000",
				Usage: "Engine serving port",
			},
			&cli.StringFlag{
				Name:  "proxy",
				Value: "",
				Usage: "Non-authenticate proxy URL for traffic redirection",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Set debug mode on runtime",
			},
		},
		Action: func(c *cli.Context) error {
			println("")
			log.Printf("%s Starting engine\n", yellow("[ Engine ]"))

			enginePort = c.String("port")
			engineProxy = c.String("proxy")
			engineDebug = c.Bool("debug")

			if engineProxy != "" {
				useProxy = true

				log.Printf("%s Using proxy %s", yellow("[ Engine ]"), engineProxy)
			}

			if engineDebug {
				log.Printf("%s Using debug mode", yellow("[ Engine ]"))
			}

			var userLauncher string

			if useProxy {
				userLauncher = launcher.New().
					Proxy(engineProxy).          // add a flag, here we set a http proxy
					Delete("use-mock-keychain"). // delete flag mock keychain
					NoSandbox(true).             // disable chromium sandbox
					Headless(!engineDebug).      // set false to debug
					MustLaunch()                 // launch the browser
			} else {
				userLauncher = launcher.New().
					NoSandbox(true).        // disable chromium sandbox
					Headless(!engineDebug). // set false to debug
					Set(`--enable-usermedia-screen-capturing`).
					Set(`--allow-http-screen-capture`).
					Set(`--disable-infobars`).
					MustLaunch() // launch the browser
			}

			log.Printf("%s Starting browser", yellow("[ Engine ]"))
			engineBrowser = *rod.New().ControlURL(userLauncher).MustConnect()

			// Start with blank page to initialize browser
			log.Printf("%s Create a blank page", yellow("[ Engine ]"))
			engineBrowser.MustPage("about:blank")

			log.Printf("%s Ready to handle scraper\n\n", yellow("[ Engine ]"))
			HandleHTTPRequest()

			return nil
		},
	}

	err := app.Run(os.Args)

	if err != nil {
		log.Fatal(err)
	}
}

// TODO Comment
// ....
func HandleHTTPRequest() {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir(resourcesDirectory))))

	http.HandleFunc("/", HandleMultiPages)
	http.HandleFunc("/favicon.ico", Noop)

	listener, errorListener := net.Listen("tcp4", ":"+enginePort)

	strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	if errorListener != nil {
		log.Printf(red("[ Engine ] %v"), errorListener)
	}

	log.Printf("%s Server running on http://127.0.0.1:%s\n", green("[ Engine ]"), enginePort)
	log.Printf("%s Waiting for connection\n\n", green("[ Engine ]"))

	sign := make(chan os.Signal)

	signal.Notify(sign, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)

	go func() {
		<-sign
		engineBrowser.MustClose()
		os.Exit(1)
	}()

	panic(http.Serve(listener, nil))
}

// TODO Comment
// ....
func HandleMultiPages(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}
	unique := uuid.New().String()
	pageId := unique[len(unique)-12:]
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	switch r.Method {
	case "POST":

		// Declare a new Person struct.
		var request types.Config

		// Try to decode the request body into the struct. If there is an error,
		// respond to the client with the error message and a 400 status code.
		err := json.NewDecoder(r.Body).Decode(&request)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("--- Create flow #%s - %s\n\n", green(pageId), green(request.Name))

		rootChannel := make(chan interface{})

		go func(rootChannel chan interface{}) {
			log.Printf("%s Flow ID : %s", yellow("[ Engine ]"), pageId)
			log.Printf("%s Flow name : %s", yellow("[ Engine ]"), request.Name)
			log.Printf("%s Flow target : %s\n\n", yellow("[ Engine ]"), request.Target)
			log.Printf("%s Starting flow", yellow("[ Engine ]"))

			if len(request.Flow) > 0 {
				start := time.Now()
				page := engineBrowser.MustPage()

				// Enable screencast frame when user use record parameter
				frameCount := 0
				diskUsage := make(map[string]float64)
				bandwidthUsage := make(map[string]float64)

				go page.EachEvent(func(e *proto.PageScreencastFrame) {
					temporaryFilePath := videoDirectory + pageId + "-" + strconv.Itoa(frameCount) + "-frame.jpeg"

					_ = utils.OutputFile(temporaryFilePath, e.Data)

					proto.PageScreencastFrameAck{
						SessionID: e.SessionID,
					}.Call(page)
					frameCount++
				}, func(e *proto.NetworkResponseReceived) {
					bandwidthUsage[strings.ToLower(string(e.Type))] += e.Response.EncodedDataLength
				})()

				if request.Record {
					quality := int(100)
					everyNthFrame := int(1)

					proto.PageStartScreencast{
						Format:        "jpeg",
						Quality:       &quality,
						EveryNthFrame: &everyNthFrame,
					}.Call(page)
				}

				htmlResult := make(map[int]map[string]string)
				screenshotResult := make(map[string]string)
				videoResult := ""

				pageRepeated := 1

				if request.Paginate && request.Repeat > 0 {
					pageRepeated = request.Repeat
				}

				environmentRepetition := os.Getenv(`MAXIMUM_REPETITION`)

				if environmentRepetition != "" {
					maximumRepetition, _ := strconv.Atoi(environmentRepetition)

					if pageRepeated > maximumRepetition {
						log.Printf("%s Repeated parameter more than ENV want %d have %d", yellow("[ Engine ]"), maximumRepetition, pageRepeated)

						pageRepeated = maximumRepetition
					}
				}

				isFinish := HandleRepeatLoop(request, request.Flow, 1, len(request.Flow), page, pageId, 0, pageRepeated, htmlResult, screenshotResult, diskUsage)

				if isFinish {
					proto.PageStopScreencast{}.Call(page)
					page.MustClose()
				}

				// Delay two second
				time.Sleep(1 * time.Second)

				if request.Record {
					_, videoPath := HandleRenderVideo(request.Name, pageId)

					pathReplacer := strings.NewReplacer(rootDirectory, "", "//", "/")
					pathReplaced := pathReplacer.Replace(string(videoPath))

					videoResult = pathReplaced

					time.Sleep(1 * time.Second)

					fileSize, errorFileSize := os.Stat(rootDirectory + pathReplaced)

					if errorFileSize != nil {
						log.Printf(red("[ Engine ] %v"), errorFileSize)
					} else {
						diskUsage["video"] += float64(fileSize.Size())
					}
				}

				slugName := slug.Make(request.Name)
				sluggableName := slugName + "-" + pageId

				resultJson := types.Response{
					Id:       pageId,
					Name:     request.Name,
					Slug:     sluggableName,
					Target:   request.Target,
					Engine:   request.Engine,
					Record:   request.Record,
					Repeat:   pageRepeated,
					Paginate: request.Paginate,
					Duration: time.Since(start) / 1000000, // milisecond
					Usage: types.ResponseUsage{
						Bandwidth: bandwidthUsage,
						Disk:      diskUsage,
					},
					Code: 200,
				}

				if len(htmlResult) > 0 {
					if len(htmlResult[0]) > 0 {
						resultJson.Result = htmlResult
					}
				}

				if len(screenshotResult) > 0 {
					resultJson.Screenshot = screenshotResult
				}

				if videoResult != "" {
					resultJson.Recording = videoResult
				}

				rootChannel <- resultJson
			} else {
				resultJson := types.Response{
					Code:    200,
					Message: "Flow not found for " + pageId,
				}

				rootChannel <- resultJson
			}
		}(rootChannel)

		result := <-rootChannel

		HandleResponse(w, result, pageId)

		log.Printf("%s Flow closed\n\n", yellow("[ Engine ]"))
	default:
		resultJson := types.Response{
			Code:    403,
			Message: "Method not allowed for this request",
		}

		HandleResponse(w, resultJson, "")
	}
}

// TODO Comment
// ....
func HandleRepeatLoop(request types.Config, flow []types.Flow, current int, total int, page *rod.Page, pageId string, pageIndex int, pageMustRepeat int, htmlResult map[int]map[string]string, screenshotResult map[string]string, diskUsage map[string]float64) bool {
	if pageIndex < pageMustRepeat {
		htmlResult[pageIndex] = make(map[string]string)

		var allowToNavigate bool = true

		if pageIndex > 0 && request.Paginate {
			allowToNavigate = false
		}

		if allowToNavigate {
			page.Navigate(request.Target)
		}

		isFinish := HandleFlowLoop(request, request.Flow, 0, len(request.Flow), page, pageId, pageIndex, htmlResult, screenshotResult, diskUsage)

		if isFinish {
			return HandleRepeatLoop(request, request.Flow, 0, len(request.Flow), page, pageId, pageIndex+1, pageMustRepeat, htmlResult, screenshotResult, diskUsage)
		} else {
			return false
		}
	}

	if pageIndex == pageMustRepeat {
		return true
	}

	return false
}

func HandleFlowLoop(request types.Config, flow []types.Flow, current int, total int, page *rod.Page, pageId string, pageIndex int, htmlResult map[int]map[string]string, screenshotResult map[string]string, diskUsage map[string]float64) bool {
	red := color.New(color.FgRed).SprintFunc()

	if current < total {
		flowData := flow[current]

		var hasElement bool = false
		var detectedElement *rod.Element

		// TODO Comment
		// ....

		if flowData.Form.Selector != "" {
			hasElement = true
		}

		// TODO Comment
		// ....

		if hasElement {
			_, element, errorMessage := page.Has(flowData.Form.Selector)

			if errorMessage != nil {
				log.Println(red("[ Engine ] element " + flowData.Form.Selector + " not found"))
			}

			detectedElement = element
		}

		if flowData.Delay != 0 {

			// TODO Comment
			// ....

			var sleepTime int = int(flowData.Delay)
			time.Sleep(time.Second * time.Duration(sleepTime))

		} else if flowData.Navigate != "" {

			// TODO Comment
			// ....

			err := rod.Try(func() {
				page.Timeout(defaultTimeout).MustElementR("a", flowData.Navigate).MustClick()
			})

			if errors.Is(err, context.DeadlineExceeded) {
				log.Println(red("[ Engine ] Anchor " + flowData.Navigate + " not found"))
			}

		} else if flowData.Form.Fill != "" {

			// TODO Comment
			// ....

			if strings.Contains(flowData.Form.Fill, "$") {
				detectedElement.MustInput(os.Getenv(strings.ReplaceAll(flowData.Form.Fill, "$", "")))
			} else {
				detectedElement.MustInput(flowData.Form.Fill)
			}

		} else if flowData.Form.Do == "Enter" {

			// TODO Comment
			// ....

			detectedElement.MustPress(input.Enter)

		} else if flowData.Form.Do == "Click" {

			// TODO Comment
			// ....

			detectedElement.MustClick()

			err := rod.Try(func() {
				page.Timeout(defaultTimeout).MustElement("body").MustWaitLoad()
			})

			if errors.Is(err, context.DeadlineExceeded) {
				log.Println(red("[ Engine ] Can't wait for body loaded"))
			}

		} else if flowData.Screenshot.Path != "" {

			// TODO Comment
			// ....

			screenshotPath := screenshotDirectory + pageId + "-" + strconv.Itoa(pageIndex) + "-" + flowData.Screenshot.Path

			if flowData.Screenshot.Clip.Top != 0 || flowData.Screenshot.Clip.Left != 0 || flowData.Screenshot.Clip.Width != 0 || flowData.Screenshot.Clip.Height != 0 {

				// TODO Comment
				// ....

				image, _ := page.Screenshot(true, &proto.PageCaptureScreenshot{
					Format:  proto.PageCaptureScreenshotFormatJpeg,
					Quality: gson.Int(100),
					Clip: &proto.PageViewport{
						X:      flowData.Screenshot.Clip.Top,
						Y:      flowData.Screenshot.Clip.Left,
						Width:  flowData.Screenshot.Clip.Width,
						Height: flowData.Screenshot.Clip.Height,
						Scale:  1,
					},
					FromSurface: true,
				})

				_ = utils.OutputFile(screenshotPath, image)
			} else {

				// TODO Comment
				// ....

				page.MustScreenshot(screenshotPath)
			}

			pathReplacer := strings.NewReplacer(rootDirectory, "", "//", "/")
			pathReplaced := pathReplacer.Replace(string(screenshotPath))

			screenshotResult[strconv.Itoa(pageIndex)+"-"+flowData.Screenshot.Path] = pathReplaced

			time.Sleep(1 * time.Second)

			fileSize, errorFileSize := os.Stat(rootDirectory + pathReplaced)

			if errorFileSize != nil {
				log.Printf(red("[ Engine ] %v"), errorFileSize)
			} else {
				diskUsage["screenshot"] += float64(fileSize.Size())
			}

		} else if len(flowData.Take) > 0 {

			// TODO Comment
			// ....

			HandleTakeLoop(flowData.Take, 0, len(flowData.Take), page, pageId, pageIndex, htmlResult)

		} else {
			// noop
		}

		return HandleFlowLoop(request, flow, current+1, total, page, pageId, pageIndex, htmlResult, screenshotResult, diskUsage)
	}

	if current == total {
		return true
	}

	return false
}

// TODO Comment
// ....

func HandleTakeLoop(take []types.Element, current int, total int, page *rod.Page, pageId string, pageIndex int, htmlResult map[int]map[string]string) bool {
	red := color.New(color.FgRed).SprintFunc()

	if current < total {
		var takeData = take[current]
		var fieldName string = takeData.Name
		var hasElement bool = false
		var detectedElement rod.Element

		fieldError := rod.Try(func() {
			if takeData.Selector != "" {
				detectedElement = *page.Timeout(defaultTimeout).MustElement(takeData.Selector)
				hasElement = true
			}

			if takeData.Contains.Selector != "" {
				detectedElement = *page.Timeout(defaultTimeout).MustElementR(takeData.Contains.Selector, takeData.Contains.Identifier)
				hasElement = true
			}

			if takeData.NextToSelector != "" {
				detectedElement = *page.Timeout(defaultTimeout).MustElement(takeData.NextToSelector).MustNext()
				hasElement = true
			}

			if takeData.NextToContains.Selector != "" {
				detectedElement = *page.Timeout(defaultTimeout).MustElementR(takeData.NextToContains.Selector, takeData.NextToContains.Identifier).MustNext()
				hasElement = true
			}
		})

		if errors.Is(fieldError, context.DeadlineExceeded) {
			log.Println(red("[ Engine ] element " + fieldName + " not found"))
		} else if fieldError != nil {
			log.Printf(red("[ Engine ] %v"), fieldError)
		}

		if hasElement {
			if takeData.Parse == "html" {
				htmlResult[pageIndex][fieldName] = string(detectedElement.MustHTML())
			}

			if takeData.Parse == "text" {
				htmlResult[pageIndex][fieldName] = string(detectedElement.MustText())
			}

			if takeData.Parse == "image" || takeData.Parse == "anchor" {
				var sourceText string

				if takeData.Parse == "image" {
					source, _ := detectedElement.Attribute("src")

					if source != nil {
						sourceText = *source
					}
				}

				if takeData.Parse == "anchor" {
					source, _ := detectedElement.Attribute("href")

					if source != nil {
						sourceText = *source
					}
				}

				pageLocation := page.MustEval("() => window.location")
				pageDomain := pageLocation.Get("origin").String()
				fieldSource := strings.ReplaceAll(pageDomain+"/"+sourceText, "//", "/")

				htmlResult[pageIndex][fieldName] = string(fieldSource)

			}
		}

		if takeData.Table.Selector != "" {
			tableElement := page.Timeout(defaultTimeout).MustElement(takeData.Table.Selector)
			tableString := tableElement.MustHTML()
			tableToken := strings.NewReader("<html><body>" + tableString + "</body></html>")
			tableTokenizer := html.NewTokenizer(tableToken)
			tableRowCount := tableElement.MustEval("() => this.querySelectorAll('tr').length").Int()

			//                  row    column value
			tableContent := make([]map[string]string, tableRowCount)

			var tableRowCounter int = 0
			var tableColumnCounter int = 0

			tableContent = extractTable(tableTokenizer, tableContent, takeData.Table.Fields, tableRowCounter, tableColumnCounter)

			resultOfTable := tableContent[1:]

			jsonTable, _ := json.Marshal(resultOfTable)

			htmlResult[pageIndex][takeData.Table.Name] = string(jsonTable)

			if takeData.Table.WithCSV {
				rescueStdout := os.Stdout
				osReader, osWriter, _ := os.Pipe()
				os.Stdout = osWriter

				stringReader := strings.NewReader(string(jsonTable))
				data, _ := readJSON(stringReader)
				value, _ := jsonpointer.Get(data, "")
				results, _ := json2csv.JSON2CSV(value)

				printCSV(os.Stdout, results, json2csv.DotNotationStyle, false)

				osWriter.Close()

				csvResult, _ := ioutil.ReadAll(osReader)

				os.Stdout = rescueStdout

				htmlResult[pageIndex][takeData.Table.Name+"_csv"] = string(csvResult)
			}
		}

		HandleTakeLoop(take, current+1, total, page, pageId, pageIndex, htmlResult)
	}

	if current == total {
		return true
	}

	return false
}

func extractTable(tableElement *html.Tokenizer, tableContent []map[string]string, tableFields []types.ElementTableField, tableRowCounter int, tableColumnCounter int) []map[string]string {
	var isContinue bool = true
	tableRow := tableElement.Next()

	if tableRow == html.StartTagToken {
		tableData := tableElement.Token()

		if tableData.Data == "tr" {
			tableContent[tableRowCounter] = make(map[string]string)
			tableColumnCounter = 0
		}

		if tableData.Data == "td" {
			inner := tableElement.Next()

			if inner == html.TextToken {
				for _, field := range tableFields {
					if tableColumnCounter == field.Index {
						tableText := (string)(tableElement.Text())
						tableData := strings.TrimSpace(tableText)

						columnValue := tableFields[field.Index].Name

						tableContent[tableRowCounter][columnValue] = tableData
					}
				}
			}
		}
	}

	if tableRow == html.EndTagToken {
		tagElement := tableElement.Token()

		if tagElement.Data == "tr" {
			tableRowCounter++
		}

		if tagElement.Data == "td" {
			tableColumnCounter++
		}

		if tagElement.Data == "table" {
			isContinue = false
		}
	}

	if isContinue {
		return extractTable(tableElement, tableContent, tableFields, tableRowCounter, tableColumnCounter)
	} else {
		return tableContent
	}
}

// TODO Comment
// ....
func HandleResponse(w http.ResponseWriter, data interface{}, pageId string) {
	yellow := color.New(color.FgYellow).SprintFunc()

	if pageId != "" {
		log.Printf("%s Sending result", yellow("[ Engine ]"))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	fileSource, _ := json.MarshalIndent(data, "", "  ")

	pathReplacer := strings.NewReplacer(`\"`, `"`, `"[`, `[`, `]"`, `]`)
	pathReplaced := pathReplacer.Replace(string(fileSource))

	w.Write([]byte(pathReplaced))
}

// TODO Comment
// ....
func Noop(w http.ResponseWriter, r *http.Request) {}

func readJSON(r io.Reader) (interface{}, error) {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()

	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}
func printCSV(w io.Writer, results []json2csv.KeyValue, headerStyle json2csv.KeyStyle, transpose bool) error {
	csv := json2csv.NewCSVWriter(w)
	csv.HeaderStyle = headerStyle
	csv.Transpose = transpose
	if err := csv.WriteCSV(results); err != nil {
		return err
	}
	return nil
}

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func HandleRenderVideo(name string, pageId string) (string, string) {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	log.Printf("%s Render recorded video", yellow("[ Engine ]"))

	slugName := slug.Make(name)
	videoName := slugName + "-" + pageId + ".mp4"
	videoPath := videoDirectory + videoName

	go func() {
		renderer, err := mjpeg.New(videoPath, int32(1440), int32(900), 6)

		if err != nil {
			log.Printf(red("[ Engine ] %v\n"), err)
		}

		matches, err := filepath.Glob(videoDirectory + pageId + "-*-frame.jpeg")

		if err != nil {
			log.Printf(red("[ Engine ] %v\n"), err)
		}

		sort.Strings(matches)

		for _, name := range matches {
			data, err := ioutil.ReadFile(name)

			if err != nil {
				log.Printf(red("[ Engine ] %v\n"), err)
			}

			renderer.AddFrame(data)
		}

		renderer.Close()

		for _, name := range matches {
			errRemove := os.Remove(name)

			if errRemove != nil {
				log.Printf(red("[ Engine ] %v\n"), errRemove)
			}
		}

		compressedPath := strings.ReplaceAll(videoPath, ".mp4", "-compressed.mp4")

		cmd := exec.Command("ffmpeg", "-i", videoPath, "-vcodec", "h264", "-acodec", "mp2", compressedPath)
		stdout, err := cmd.Output()

		if err != nil {
			log.Printf(red("[ Engine ] %v\n"), err)
		}

		if len(stdout) > 0 {
			log.Printf("%s %v\n", yellow("[ Engine ]"), stdout)
		}

		errRemoveOriginal := os.Remove(videoPath)

		if errRemoveOriginal != nil {
			log.Printf(red("[ Engine ] %v\n"), errRemoveOriginal)
		}

		errRenameCompressed := os.Rename(compressedPath, videoPath)

		if errRenameCompressed != nil {
			log.Printf(red("[ Engine ] %v\n"), errRenameCompressed)
		}
	}()

	return videoName, videoPath
}
