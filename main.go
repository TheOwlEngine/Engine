package main

import (
	"context"
	"encoding/json"
	"engine/types"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"log"
	"net"
	"net/http"
	"os"
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
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
	"github.com/ysmood/gson"
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
var jsonDirectory string
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
	jsonDirectory = resourcesDirectory + "/json/"
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
					MustLaunch()            // launch the browser
			}

			log.Printf("%s Start browser", yellow("[ Engine ]"))
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
	green := color.New(color.FgGreen).SprintFunc()

	http.HandleFunc("/", HandleMultiPages)
	http.HandleFunc("/favicon.ico", Noop)

	listener, err := net.Listen("tcp4", ":"+enginePort)

	strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	if err != nil {
		panic(err)
	}

	log.Printf("%s Server running on http://127.0.0.1:%s\n", green("[ Engine ]"), enginePort)
	log.Printf("%s Waiting for connection\n", green("[ Engine ]"))

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
	unique := uuid.New().String()
	pageId := unique[len(unique)-12:]
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

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

		fmt.Printf("\n--- Create flow #%s - %s\n\n", green(pageId), green(request.Name))

		rootChannel := make(chan interface{})

		go func(rootChannel chan interface{}) {
			log.Printf("%s Starting flow", yellow("[ Engine ]"))

			if len(request.Flow) > 0 {

				page := engineBrowser.MustPage()

				// If website is HTML only and not rendered with JavaScript
				// let skip browser to disable screenshot the resources like
				// image, stylesheet, media, ping, font
				if request.HtmlOnly {
					router := page.HijackRequests()

					// since we are only hijacking a specific page, even using the "*" won't affect much of the performance
					router.MustAdd("*", func(ctx *rod.Hijack) {
						// use NetworkResourceTypeScript for javascript files, there're a lot of types you can use in this enum
						if ctx.Request.Type() == proto.NetworkResourceTypeImage || ctx.Request.Type() == proto.NetworkResourceTypeStylesheet || ctx.Request.Type() == proto.NetworkResourceTypeMedia || ctx.Request.Type() == proto.NetworkResourceTypePing || ctx.Request.Type() == proto.NetworkResourceTypeFont {
							ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
							return
						}

						ctx.ContinueRequest(&proto.FetchContinueRequest{})
					})

					go router.Run()
				}

				htmlString := make(map[int]map[string]string)

				pageRepeated := 1

				if request.Repeat > 0 {
					pageRepeated = request.Repeat
				}

				isFinish := HandleRepeatLoop(request, request.Flow, 1, len(request.Flow), page, pageId, 0, pageRepeated, htmlString)

				if isFinish {
					page.MustClose()
				}

				resultJson := types.Response{
					Id:     pageId,
					Name:   request.Name,
					Target: request.Target,
					Engine: request.Engine,
					Code:   200,
					Result: htmlString,
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

		log.Printf("%s Flow closed", yellow("[ Engine ]"))
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
func HandleRepeatLoop(request types.Config, flow []types.Flow, current int, total int, page *rod.Page, pageId string, pageIndex int, pageMustRepeat int, htmlString map[int]map[string]string) bool {
	if pageIndex < pageMustRepeat {
		htmlString[pageIndex] = make(map[string]string)

		var allowToNavigate bool = true

		if pageIndex > 0 && request.Paginate {
			allowToNavigate = false
		}

		if allowToNavigate {
			page.Navigate(request.Target)
		}

		isFinish := HandleFlowLoop(request, request.Flow, 0, len(request.Flow), page, pageId, pageIndex, htmlString)

		if isFinish {
			return HandleRepeatLoop(request, request.Flow, 0, len(request.Flow), page, pageId, pageIndex+1, pageMustRepeat, htmlString)
		} else {
			return false
		}
	}

	if pageIndex == pageMustRepeat {
		return true
	}

	return false
}

func HandleFlowLoop(request types.Config, flow []types.Flow, current int, total int, page *rod.Page, pageId string, pageIndex int, htmlString map[int]map[string]string) bool {
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

			screenshotPath := screenshotDirectory + pageId + "-" + flowData.Screenshot.Path

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

		} else if len(flowData.Take) > 0 {

			// TODO Comment
			// ....

			HandleTakeLoop(flowData.Take, 0, len(flowData.Take), page, pageId, pageIndex, htmlString)

		} else {
			// noop
		}

		return HandleFlowLoop(request, flow, current+1, total, page, pageId, pageIndex, htmlString)
	}

	if current == total {
		return true
	}

	return false
}

// TODO Comment
// ....

func HandleTakeLoop(take []types.Element, current int, total int, page *rod.Page, pageId string, pageIndex int, htmlString map[int]map[string]string) bool {
	red := color.New(color.FgRed).SprintFunc()

	if current < total {
		var takeData = take[current]
		var fieldName string = takeData.Name
		var fieldElement rod.Element

		err := rod.Try(func() {
			if takeData.Selector != "" {
				fieldElement = *page.Timeout(defaultTimeout).MustElement(takeData.Selector)
			}

			if takeData.Contains.Selector != "" {
				fieldElement = *page.Timeout(defaultTimeout).MustElementR(takeData.Contains.Selector, takeData.Contains.Identifier)
			}

			if takeData.NextToSelector != "" {
				fieldElement = *page.Timeout(defaultTimeout).MustElement(takeData.NextToSelector).MustNext()
			}

			if takeData.NextToContains.Selector != "" {
				fieldElement = *page.Timeout(defaultTimeout).MustElementR(takeData.NextToContains.Selector, takeData.NextToContains.Identifier).MustNext()
			}

			if takeData.Parse == "html" {
				htmlString[pageIndex][fieldName] = string(fieldElement.MustHTML())
			}

			if takeData.Parse == "text" {
				htmlString[pageIndex][fieldName] = string(fieldElement.MustText())
			}
		})

		if takeData.Table.Selector != "" {
			tableElement := page.Timeout(defaultTimeout).MustElement(takeData.Table.Selector)
			tableString := tableElement.MustHTML()
			tableToken := strings.NewReader("<html><body>" + tableString + "</body></html>")
			tableTokenizer := html.NewTokenizer(tableToken)
			tableRowCount := tableElement.MustEval("() => this.querySelectorAll('tr').length").Int()

			//                       row     column value
			tableContent := make([]map[string]string, tableRowCount)

			var tableRowCounter int = 0
			var tableColumnCounter int = 0

			tableContent = extractTable(tableTokenizer, tableContent, takeData.Table.Fields, tableRowCounter, tableColumnCounter)

			resultOfTable := tableContent[1:]

			jsonTable, _ := json.Marshal(resultOfTable)

			htmlString[pageIndex][takeData.Table.Name] = string(jsonTable)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			log.Println(red("[ Engine ] element " + fieldName + " not found"))
		} else if err != nil {
			panic(err)
		}

		HandleTakeLoop(take, current+1, total, page, pageId, pageIndex, htmlString)
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

	fileReplacer := strings.NewReplacer(`\"`, `"`, `"[`, `[`, `]"`, `]`)
	fileDecode := fileReplacer.Replace(string(fileSource))

	w.Write([]byte(fileDecode))
}

// TODO Comment
// ....
func Noop(w http.ResponseWriter, r *http.Request) {}
