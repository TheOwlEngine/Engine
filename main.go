package main

import (
	"context"
	"encoding/json"
	"engine/types"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"log"
	"net"
	"net/http"
	"net/url"
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
	"golang.org/x/net/html"
)

var owlProxyAPI string = "https://owlengine.com/api/proxy?url="

var engineBrowser rod.Browser

var enginePort string
var engineProxy string
var engineDebug bool

var useProxy bool = false

var defaultTimeout time.Duration

var rootDirectory string
var resourcesDirectory string
var imagesDirectory string
var videoDirectory string
var logsDirectory string

var temporaryDomainName string
var temporaryNavigateUrl string
var temporaryWrapperElement string
var temporaryInfiniteScroll int

var globalErrors []string

func main() {
	godotenv.Load(".env")

	defaultTimeout = 3 * time.Second

	rootDirectory, _ = os.Getwd()

	resourcesDirectory = rootDirectory + "/resources/"
	imagesDirectory = resourcesDirectory + "/images/"
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
					Set(`--enable-usermedia-screen-capturing`).
					Set(`--allow-http-screen-capture`).
					Set(`--disable-infobars`).
					MustLaunch() // launch the browser
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

	errorCliApp := app.Run(os.Args)

	if errorCliApp != nil {
		log.Fatal(errorCliApp)
	}
}

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
		globalErrors = append(globalErrors, `Something went wrong on our server`)
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

func HandleResponse(w http.ResponseWriter, data types.Result, pageId string) {
	yellow := color.New(color.FgYellow).SprintFunc()

	if pageId != "" {
		log.Printf("%s Sending result", yellow("[ Engine ]"))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	for _, page := range data.Result {
		for index := range page.Content {
			if strings.Contains(page.Content[index].Content, `[`) {
				stringReplacer := strings.NewReplacer(`"`, `$"`, "\n", "$\n")
				page.Content[index].Content = stringReplacer.Replace(page.Content[index].Content)
			}
		}
	}

	jsonTable, _ := json.Marshal(data)
	jsonEncoded := Unescape(jsonTable)
	jsonReplacer := strings.NewReplacer(`"{`, `{`, `}"`, `}`, `"[`, `[`, `]"`, `]`, `$\"`, `"`, `$\n`, "\n")
	jsonResult := jsonReplacer.Replace(jsonEncoded)

	w.Write([]byte(jsonResult))
}

func Noop(w http.ResponseWriter, r *http.Request) {}

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

	return videoName, videoPath
}

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

		var request types.Config

		// Try to decode the request body into the struct. If there is an error,
		// respond to the client with the error message and a 400 status code.
		errorDecodeRequest := json.NewDecoder(r.Body).Decode(&request)

		if errorDecodeRequest != nil {
			http.Error(w, errorDecodeRequest.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("--- Process flow for #%s - %s\n\n", green(pageId), green(request.Name))

		rootChannel := make(chan types.Result)

		go func(rootChannel chan types.Result) {

			temporaryDomainName = ""     // Clean up temporary domain name
			temporaryNavigateUrl = ""    // Clean up temporary navigate url
			temporaryWrapperElement = "" // Clean up temporary wrapper element
			temporaryInfiniteScroll = 0  // Clean up temporary infinite scroll
			globalErrors = nil           // Clean up global errors

			log.Printf("%s Flow ID : %s", yellow("[ Engine ]"), pageId)
			log.Printf("%s Flow name : %s", yellow("[ Engine ]"), request.Name)
			log.Printf("%s Flow target : %s\n\n", yellow("[ Engine ]"), request.FirstPage)
			log.Printf("%s Starting flow", yellow("[ Engine ]"))

			if len(request.Flow) > 0 {
				start := time.Now()
				page := engineBrowser.MustPage()

				proxyAddress := page.MustNavigate("https://echo.owlengine.com/ip").MustWaitLoad().MustElement("body").MustText()

				// Enable screencast frame when user use record parameter
				frameCounter := 0
				diskUsage := make(map[string]float64)
				bandwidthUsage := make(map[string]float64)

				go page.EachEvent(func(e *proto.PageScreencastFrame) {
					frameCount := "0" + strconv.Itoa(frameCounter)

					if frameCounter > 9 {
						frameCount = strconv.Itoa(frameCounter)
					}

					temporaryFilePath := videoDirectory + pageId + "-" + frameCount + "-frame.jpeg"

					_ = utils.OutputFile(temporaryFilePath, e.Data)

					proto.PageScreencastFrameAck{
						SessionID: e.SessionID,
					}.Call(page)
					frameCounter++
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

				paginateLimit := 1

				if request.Paginate && request.PaginateLimit > 0 {
					paginateLimit = request.PaginateLimit
				}

				if request.Infinite && request.InfiniteScroll > 0 {
					paginateLimit = request.InfiniteScroll
				}

				temporaryScraperResult := make([]types.ResultPage, 0, paginateLimit)
				recordResult := ""

				environmentRepetition := os.Getenv(`MAX_PAGINATE_LIMIT`)

				if environmentRepetition != "" {
					maximumRepetition, _ := strconv.Atoi(environmentRepetition)

					if paginateLimit > maximumRepetition {
						log.Printf("%s Limit parameter more than ENV want %d have %d", yellow("[ Engine ]"), maximumRepetition, paginateLimit)

						paginateLimit = maximumRepetition
					}
				}

				if request.ItemsOnPage > 0 {
					paginateLimit = request.ItemsOnPage * paginateLimit
				}

				parsedUrl, errorParseUrl := url.Parse(request.FirstPage)

				if errorParseUrl != nil {
					log.Printf(red("[ Engine ] %v"), errorParseUrl)
					globalErrors = append(globalErrors, `Failed to decode your first page URL`)
				}

				temporaryDomainName = parsedUrl.Scheme + "://" + parsedUrl.Hostname()

				isFinish, scraperResult := HandleRepeatLoop(request, request.Flow, page, pageId, 0, paginateLimit, temporaryScraperResult, diskUsage)

				if isFinish {
					proto.PageStopScreencast{}.Call(page)
					page.MustClose()
				}

				time.Sleep(2 * time.Second)

				if request.Record {
					_, videoPath := HandleRenderVideo(request.Name, pageId)

					pathReplacer := strings.NewReplacer(rootDirectory, "", "//", "/")
					pathReplaced := pathReplacer.Replace(string(videoPath))

					recordResult = pathReplaced

					time.Sleep(1 * time.Second)

					fileSize, errorFileSize := os.Stat(rootDirectory + pathReplaced)

					if errorFileSize != nil {
						log.Printf(red("[ Engine ] %v"), errorFileSize)
						globalErrors = append(globalErrors, `Failed to read recorded video size`)
					} else {
						diskUsage["videos"] += float64(fileSize.Size())
					}
				}

				slugName := slug.Make(request.Name)
				sluggableName := slugName + "-" + pageId

				resultJson := types.Result{
					Id:             pageId,
					Proxy:          proxyAddress,
					Code:           200,
					Name:           request.Name,
					Slug:           sluggableName,
					Message:        "The flow is running successfully",
					Duration:       time.Since(start) / 1000000, // milisecond,
					Engine:         string(request.Engine),
					FirstPage:      string(request.FirstPage),
					ItemsOnPage:    request.ItemsOnPage,
					Infinite:       request.Infinite,
					InfiniteScroll: request.InfiniteScroll,
					Paginate:       request.Paginate,
					PaginateLimit:  request.PaginateLimit,
					Record:         request.Record,
					Usage: types.ResultUsage{
						Bandwidth: bandwidthUsage,
						Disk:      diskUsage,
					},
					Errors: globalErrors,
				}

				if len(scraperResult) > 0 {
					resultJson.Result = scraperResult
				}

				if recordResult != "" {
					resultJson.Recording = owlProxyAPI + recordResult
				}

				rootChannel <- resultJson
			} else {
				resultJson := types.Result{
					Code:    404,
					Message: "Flow not found for " + pageId,
				}

				rootChannel <- resultJson
			}
		}(rootChannel)

		result := <-rootChannel

		HandleResponse(w, result, pageId)

		log.Printf("%s Flow closed\n\n", yellow("[ Engine ]"))
	default:
		resultJson := types.Result{
			Code:    400,
			Message: "Method not allowed for this request",
		}

		HandleResponse(w, resultJson, "")
	}
}

func HandleRepeatLoop(request types.Config, flow []types.Flow, page *rod.Page, pageId string, paginateIndex int, paginateLimit int, scraperResult []types.ResultPage, diskUsage map[string]float64) (bool, []types.ResultPage) {
	red := color.New(color.FgRed).SprintFunc()

	pageStart := time.Now()
	temporaryContents := make([]types.ResultContent, 0, len(request.Flow))

	if paginateIndex == 0 {
		err := rod.Try(func() {
			page.Timeout(10 * time.Second).MustNavigate(request.FirstPage)
			page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
			page.MustWaitLoad()
		})

		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf(red("[ Engine ] Failed to navigate to %s, due to context deadline exceeded"), request.FirstPage)
		} else if err != nil {
			log.Printf(red("[ Engine ] Failed to navigate to %s, due to %v"), request.FirstPage, err)
		}
	}

	if request.ItemsOnPage > 0 && paginateLimit > 0 {
		if paginateIndex >= request.ItemsOnPage && paginateIndex%request.ItemsOnPage == 0 && paginateIndex < paginateLimit {
			if request.PaginateButton != "" {
				page.MustElement(request.PaginateButton).MustClick()
			}

			if request.Infinite && temporaryInfiniteScroll < request.InfiniteScroll {
				page.Mouse.Scroll(0, float64(*page.MustGetWindow().Height)*4, 2)
				temporaryInfiniteScroll++
			}

			page.MustWaitLoad()
			time.Sleep(defaultTimeout)
		}
	}

	if paginateIndex < paginateLimit {

		isFinish, pageContent := HandleFlowLoop(request, request.Flow, 0, len(request.Flow), page, pageId, paginateIndex, temporaryContents, diskUsage)

		if isFinish {
			scraperResult = append(scraperResult, types.ResultPage{
				Title:    page.MustInfo().Title,
				Url:      page.MustInfo().URL,
				Page:     paginateIndex + 1,
				Duration: time.Since(pageStart) / 1000000,
				Content:  pageContent,
			})

			return HandleRepeatLoop(request, request.Flow, page, pageId, paginateIndex+1, paginateLimit, scraperResult, diskUsage)
		} else {
			return false, scraperResult
		}
	}

	if paginateIndex == paginateLimit {
		return true, scraperResult
	}

	return false, scraperResult
}

func HandleFlowLoop(request types.Config, flow []types.Flow, current int, total int, page *rod.Page, pageId string, paginateIndex int, pageContent []types.ResultContent, diskUsage map[string]float64) (bool, []types.ResultContent) {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	if current < total {
		flowData := flow[current]

		var fieldName string = ""
		var detectedElement *rod.Element
		var selectorText string
		var resultContent types.ResultContent

		var selectorStringReplacer = strings.NewReplacer(`"`, `'`, `[`, ``, `]`, ``)

		currentItemIndex := paginateIndex - (request.ItemsOnPage * int(math.Floor(float64(paginateIndex)/float64(request.ItemsOnPage))))

		if flowData.Wrapper != "" {
			temporaryWrapperElement = flowData.Wrapper
		}

		if flowData.Element.Selector != "" {
			selectorText = flowData.Element.Selector
		}

		if flowData.Element.Contains.Selector != "" {
			selectorText = flowData.Element.Contains.Selector
		}

		if flowData.Capture.Name != "" {
			if flowData.Capture.Selector != "" {
				selectorText = flowData.Capture.Selector
			} else {
				selectorText = "body"
			}

			fieldName = flowData.Capture.Name
		}

		if flowData.Take.Selector != "" {
			selectorText = flowData.Take.Selector
			fieldName = flowData.Take.Name
		}

		if flowData.Take.NextToSelector != "" {
			selectorText = flowData.Take.NextToSelector
			fieldName = flowData.Take.Name
		}

		if flowData.Take.Contains.Selector != "" {
			selectorText = flowData.Take.Contains.Selector
			fieldName = flowData.Take.Name
		}

		if flowData.Take.NextToContains.Selector != "" {
			selectorText = flowData.Take.NextToContains.Selector
			fieldName = flowData.Take.Name
		}

		if flowData.Table.Selector != "" {
			selectorText = flowData.Table.Selector
			fieldName = flowData.Table.Name
		}

		if flowData.WaitFor.Selector != "" {
			selectorText = flowData.Table.Selector
		}

		if temporaryWrapperElement != "" {
			selectorText = temporaryWrapperElement + " " + selectorText
		}

		if strings.Contains(selectorText, "$loop_index") {
			selectorText = strings.ReplaceAll(selectorText, "$loop_index", strconv.Itoa(paginateIndex))
		}

		if strings.Contains(selectorText, "$loop_number") {
			selectorText = strings.ReplaceAll(selectorText, "$loop_number", strconv.Itoa(paginateIndex+1))
		}

		if strings.Contains(selectorText, "$item_index") {
			selectorText = strings.ReplaceAll(selectorText, "$item_index", strconv.Itoa(currentItemIndex))
		}

		if strings.Contains(selectorText, "$item_number") {
			selectorText = strings.ReplaceAll(selectorText, "$item_number", strconv.Itoa(currentItemIndex+1))
		}

		fieldError := rod.Try(func() {
			if flowData.Element.Selector != "" || flowData.Table.Selector != "" || flowData.Take.Selector != "" || flowData.Capture.Name != "" {
				detectedElement = page.Timeout(defaultTimeout).MustElement(selectorText)
			} else if flowData.Element.Contains.Selector != "" {
				detectedElement = page.Timeout(defaultTimeout).MustElementR(selectorText, flowData.Element.Contains.Identifier)
			} else if flowData.Take.Contains.Selector != "" {
				detectedElement = page.Timeout(defaultTimeout).MustElementR(selectorText, flowData.Take.Contains.Identifier)
			} else if flowData.Take.NextToSelector != "" {
				detectedElement = page.Timeout(defaultTimeout).MustElement(selectorText).MustNext()
			} else if flowData.Take.NextToContains.Selector != "" {
				detectedElement = page.Timeout(defaultTimeout).MustElementR(selectorText, flowData.Take.NextToContains.Identifier).MustNext()
			}
		})

		if flowData.Element.Contains.Identifier != "" {
			selectorText = selectorText + " `" + flowData.Element.Contains.Identifier + "`"
		}

		if flowData.Take.Contains.Identifier != "" {
			selectorText = selectorText + " `" + flowData.Take.Contains.Identifier + "`"
		}

		if flowData.Take.NextToContains.Identifier != "" {
			selectorText = selectorText + " `" + flowData.Take.NextToContains.Identifier + "`"
		}

		if errors.Is(fieldError, context.DeadlineExceeded) {
			log.Printf(red("[ Engine ] Selector %s not found"), selectorText)
			globalErrors = append(globalErrors, fmt.Sprintf(`Failed to find selector %s for %s`, selectorStringReplacer.Replace(selectorText), fieldName))
		} else if fieldError != nil {
			log.Printf(red("[ Engine ] %v"), fieldError)
			globalErrors = append(globalErrors, fmt.Sprintf(`Failed to find selector %s for %s`, selectorStringReplacer.Replace(selectorText), fieldName))
		}

		// Process without Element

		if flowData.Delay != 0 {

			var sleepTime int = int(flowData.Delay)
			time.Sleep(time.Second * time.Duration(sleepTime))

		} else if flowData.WaitFor.Selector != "" {

			var waitTimeOut = 10 * time.Second

			if flowData.WaitFor.Delay > 0 {
				var sleepTime int = int(flowData.WaitFor.Delay)
				waitTimeOut = time.Second * time.Duration(sleepTime)
			}

			err := rod.Try(func() {
				page.Timeout(waitTimeOut).MustElement(selectorText)
				page.MustWaitLoad()
			})

			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf(red("[ Engine ] Failed to wait for selector %s, due to context deadline exceeded"), selectorText)
				globalErrors = append(globalErrors, fmt.Sprintf(`Failed to wait for selector %s`, selectorStringReplacer.Replace(selectorText)))
			} else if err != nil {
				log.Printf(red("[ Engine ] Failed to wait for selector %s, due to %v"), selectorText, err)
				globalErrors = append(globalErrors, fmt.Sprintf(`Failed to wait for selector %s`, selectorStringReplacer.Replace(selectorText)))
			}

		} else if flowData.Scroll > 0 {

			page.Mouse.Scroll(0, float64(*page.MustGetWindow().Height), flowData.Scroll)

		} else if flowData.Navigate {

			if temporaryNavigateUrl != "" {
				temporaryWrapperElement = ""

				log.Printf(yellow("[ Engine ] Page Index %d"), paginateIndex)
				log.Printf(yellow("[ Engine ] Navigate Url %s"), temporaryNavigateUrl)

				err := rod.Try(func() {
					page.Timeout(10 * time.Second).MustNavigate(temporaryNavigateUrl)
					page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
					page.MustWaitLoad()
				})

				if errors.Is(err, context.DeadlineExceeded) {
					log.Printf(red("[ Engine ] Failed to navigate to %s, due to context deadline exceeded"), request.FirstPage)
				} else if err != nil {
					log.Printf(red("[ Engine ] Failed to navigate to %s, due to %v"), request.FirstPage, err)
				}
			}

		} else if flowData.BackToPrevious {

			temporaryWrapperElement = ""

			page.MustNavigateBack()
			page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
			page.MustWaitLoad()

		}

		// Process with Element

		if detectedElement != nil {

			if flowData.Element.Write != "" {

				if strings.Contains(flowData.Element.Write, "$") {
					detectedElement.MustInput(os.Getenv(strings.ReplaceAll(flowData.Element.Write, "$", "")))
				} else {
					detectedElement.MustInput(flowData.Element.Write)
				}

			} else if flowData.Element.Value != "" {

				detectedElement.Eval("() => this.value = " + flowData.Element.Value)

			} else if flowData.Element.Select != "" {

				detectedElement.MustSelect(flowData.Element.Select)

			} else if len(flowData.Element.Multiple) > 0 {

				for _, value := range flowData.Element.Multiple {
					detectedElement.MustSelect(value)
				}

			} else if flowData.Capture.Name != "" {

				capturePath := imagesDirectory + pageId + "-" + strconv.Itoa(paginateIndex) + "-" + flowData.Capture.Name + ".jpeg"
				captureOptions := &proto.PageCaptureScreenshot{
					Format:      proto.PageCaptureScreenshotFormatJpeg,
					Quality:     gson.Int(100),
					FromSurface: true,
				}

				if flowData.Capture.Clip.Top != 0 || flowData.Capture.Clip.Left != 0 || flowData.Capture.Clip.Width != 0 || flowData.Capture.Clip.Height != 0 {
					captureOptions.Clip = &proto.PageViewport{
						X:      flowData.Capture.Clip.Top,
						Y:      flowData.Capture.Clip.Left,
						Width:  flowData.Capture.Clip.Width,
						Height: flowData.Capture.Clip.Height,
						Scale:  1,
					}
				}

				if selectorText == "body" {
					image, _ := page.Screenshot(true, captureOptions)

					_ = utils.OutputFile(capturePath, image)
				} else {
					captureError := rod.Try(func() {
						detectedElement.MustScreenshot(capturePath)
					})

					if captureError != nil {
						log.Printf(red("%s Failed to capture missing element %s"), "[ Engine ]", flowData.Capture.Selector)
						globalErrors = append(globalErrors, fmt.Sprintf(`Failed to capture missing selector %s for %s`, selectorStringReplacer.Replace(flowData.Capture.Selector), flowData.Capture.Name))
					}
				}

				pathReplacer := strings.NewReplacer(rootDirectory, "", "//", "/")
				pathReplaced := pathReplacer.Replace(string(capturePath))
				pathRelative := rootDirectory + pathReplaced

				time.Sleep(1 * time.Second)

				fileSize := 0

				filePosition, errorFilePosition := os.Stat(pathRelative)

				if errorFilePosition != nil {
					log.Printf(red("[ Engine ] %v"), errorFilePosition)
				} else {
					fileSize = int(filePosition.Size())
				}

				diskUsage["images"] += float64(fileSize)

				resultContent.Type = "image"
				resultContent.Length = fileSize
				resultContent.Name = flowData.Capture.Name

				if fileSize > 0 {
					resultContent.Content = owlProxyAPI + pathReplaced
				} else {
					resultContent.Content = ""
				}

			} else if flowData.Element.Check != "" || flowData.Element.Radio != "" || flowData.Element.Action == "Click" {

				detectedElement.MustClick()
				page.MustWaitLoad()

				// } else if flowData.Element.Upload != "" {

				// 	detectedElement.MustSetFiles(flowData.Element.Upload)

			} else if flowData.Element.Action == "Enter" {

				detectedElement.MustPress(input.Enter)

			} else if flowData.Take.Parse != "" {

				if flowData.Take.Parse == "html" {

					resultContent.Type = "html"
					resultContent.Length = len(detectedElement.MustHTML())
					resultContent.Name = fieldName
					resultContent.Content = string(detectedElement.MustHTML())

				}

				if flowData.Take.Parse == "text" {

					resultContent.Type = "text"
					resultContent.Length = len(string(detectedElement.MustText()))
					resultContent.Name = fieldName
					resultContent.Content = string(detectedElement.MustText())

				}

				if flowData.Take.Parse == "image" || flowData.Take.Parse == "anchor" {
					var sourceText string

					if flowData.Take.Parse == "image" {
						source, _ := detectedElement.Attribute("src")

						if source != nil {
							sourceText = *source
						}
					}

					if flowData.Take.Parse == "anchor" {
						source, _ := detectedElement.Attribute("href")

						if source != nil {
							sourceText = *source

							if !strings.Contains(sourceText, "http") {
								sourceText = strings.ReplaceAll(temporaryDomainName+sourceText, "//", "/")
							}
						}

						if flowData.Take.UseForNavigate {
							temporaryNavigateUrl = sourceText
						}
					}

					resultContent.Type = flowData.Take.Parse
					resultContent.Length = len(sourceText)
					resultContent.Name = fieldName
					resultContent.Content = string(sourceText)
				}

			} else if flowData.Table.Name != "" {

				var tableHeader []types.ResultTableHead
				var tableRow [][]types.ResultTableData

				var tableRowCount int = 0
				var tableCellCount int = 0
				var tableHeaderCount int = 0

				var tableCellName string
				var tableCellType string
				var tableCellContent string

				var temporaryCellName string
				var temporaryTableData []types.ResultTableData
				var temporaryTableHeader []string
				var temporaryTableAnchor bool
				var temporaryTableHyperlink bool

				tableString := detectedElement.MustHTML()

				walk := html.NewTokenizer(strings.NewReader(`<html><body>` + tableString + `</body></html>`))

				for walk.Token().Data != "html" {
					regexNewline := regexp.MustCompile(`\r?\n|\t`)
					regexSpaces := regexp.MustCompile(`\s\s+`)

					token := walk.Next()
					data := walk.Token()
					attribute := data.Attr
					content := regexNewline.ReplaceAllString(data.Data, "")
					content = regexSpaces.ReplaceAllString(content, " ")

					if token == html.StartTagToken {
						if content == "tr" {
							tableCellCount = 0
						}

						if content == "td" || content == "th" {
							tableCellCount++
						}
					}

					if token == html.EndTagToken {

						if content == "thead" || content == "tr" {
							tableRowCount++
						}

						if content == "table" {
							break
						}

					}

					if tableRowCount == 0 {
						continueExtract := true

						if tableCellCount > 0 {
							if token == html.TextToken && content != "" && content != " " && len(content) > 0 {
								temporaryTableHeader = append(temporaryTableHeader, content)

								if len(flowData.Table.Fields) > 0 {
									continueExtract = Contains(flowData.Table.Fields, content)
								}

								if continueExtract {
									tableHeader = append(tableHeader, types.ResultTableHead{
										Index:   tableCellCount,
										Length:  len(content),
										Content: content,
									})
								}
							}
						}

						tableHeaderCount = tableCellCount
					}

					if tableRowCount > 0 {
						continueExtract := true

						if token == html.StartTagToken && content == "tr" {
							temporaryTableData = make([]types.ResultTableData, 0)
						}

						if token == html.StartTagToken && (content == "td" || content == "th") {
							tableCellType = "text"
							tableCellContent = ""

							for cellIndex := range temporaryTableHeader {
								if cellIndex == tableCellCount-1 {
									tableCellName = temporaryTableHeader[tableCellCount-1]
									temporaryCellName = temporaryTableHeader[tableCellCount-1]
								}
							}

							temporaryTableAnchor = false
							temporaryTableHyperlink = false
						}

						if token == html.StartTagToken && content == "a" {
							tableCellType = "anchor"

							for _, attr := range attribute {
								if attr.Key == "href" {
									tableCellName = attr.Val
									temporaryTableHyperlink = true
								}
							}

							temporaryTableAnchor = true
						}

						if token == html.StartTagToken && content == "img" {
							tableCellType = "image"

							for _, attr := range attribute {
								if attr.Key == "alt" {
									tableCellName = attr.Val
								}
								if attr.Key == "src" {
									tableCellContent = attr.Val
									temporaryTableHyperlink = true
								}
							}
						}

						if len(tableCellContent) == 0 && token == html.TextToken && content != "" && content != " " && len(content) > 0 {

							if temporaryTableAnchor {
								temporaryName := tableCellName

								tableCellName = content
								tableCellContent = temporaryName
							} else {
								tableCellContent = content
							}
						}

						if !temporaryTableAnchor && token == html.TextToken && tableCellContent != content {
							tableCellContent = tableCellContent + " " + content
						}

						if token == html.EndTagToken && (content == "td" || content == "th") {
							if len(flowData.Table.Fields) > 0 {
								continueExtract = Contains(flowData.Table.Fields, temporaryCellName)
							}

							if len(tableCellContent) == 0 {
								tableCellType = "number"
								tableCellContent = strconv.Itoa(tableRowCount)
							}

							if len(tableCellContent) > 0 && continueExtract {
								if temporaryTableHyperlink && !strings.Contains(tableCellContent, "http") {
									tableCellContent = strings.ReplaceAll(temporaryDomainName+tableCellContent, "//", "/")
								}

								temporaryTableData = append(temporaryTableData, types.ResultTableData{
									Type:    tableCellType,
									Index:   tableCellCount,
									Length:  len(tableCellContent),
									Name:    tableCellName,
									Content: tableCellContent,
								})
							}
						}

						if token == html.EndTagToken && content == "tr" {
							if temporaryTableData != nil {
								tableRow = append(tableRow, temporaryTableData)
							}

							temporaryTableData = nil
						}
					}

				}

				jsonTable, _ := json.Marshal(types.ResultTable{
					Name:   flowData.Table.Name,
					Column: tableHeaderCount,
					Row:    tableRowCount - 1,
					Header: tableHeader,
					Data:   tableRow,
				})

				resultContent.Type = "table"
				resultContent.Length = len(jsonTable)
				resultContent.Name = fieldName
				resultContent.Content = string(jsonTable)
			}
		}

		if resultContent.Content != "" {
			pageContent = append(pageContent, resultContent)
		}

		return HandleFlowLoop(request, flow, current+1, total, page, pageId, paginateIndex, pageContent, diskUsage)
	}

	if current == total {
		return true, pageContent
	}

	return false, pageContent
}
