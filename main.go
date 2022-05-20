package main

import (
	"encoding/json"
	"engine/types"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/DrSmithFr/go-console/pkg/style"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/google/uuid"
	"github.com/urfave/cli"
	"github.com/ysmood/gson"
)

// var engineSession string
var engineBrowser rod.Browser

var enginePort string
var engineProxy string
var engineDebug bool

var useProxy bool = false

var rootDirectory string
var resourcesDirectory string
var downloadDirectory string
var jsonDirectory string
var logsDirectory string

func main() {
	styler := style.NewConsoleStyler()

	// enable stylish errors
	defer styler.HandleRuntimeException()

	rootDirectory, _ = os.Getwd()

	resourcesDirectory = rootDirectory + "/resources/"
	downloadDirectory = resourcesDirectory + "/download/"
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

				log.Printf("[ Engine ] Using proxy %s", engineProxy)
				styler.Note("[ Engine ] Using proxy " + engineProxy)
			}

			if engineDebug {
				log.Printf("[ Engine ] Using debug mode")
				styler.Note("[ Engine ] Using debug mode")
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

			engineBrowser = *rod.New().ControlURL(userLauncher).MustConnect()

			// Start with blank page to initialize browser
			log.Println("[ Engine ] Create a blank page")
			engineBrowser.MustPage("about:blank")

			log.Println("[ Engine ] Ready to handle multi-pages scraper")
			HandleHTTPRequest(styler)

			return nil
		},
	}

	err := app.Run(os.Args)

	if err != nil {
		log.Fatal(err)
	}
}

func HandleHTTPRequest(styler *style.GoStyler) {
	http.HandleFunc("/", HandleMultiPages)
	http.HandleFunc("/favicon.ico", Noop)

	listener, err := net.Listen("tcp4", ":"+enginePort)

	strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	if err != nil {
		panic(err)
	}

	log.Printf("[ Engine ] Running on port %s", enginePort)
	styler.Success("[ Engine ] Running on http://127.0.0.1:" + enginePort)

	sign := make(chan os.Signal)

	signal.Notify(sign, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)

	go func() {
		<-sign
		engineBrowser.MustClose()
		os.Exit(1)
	}()

	panic(http.Serve(listener, nil))
}

func HandleMultiPages(w http.ResponseWriter, r *http.Request) {
	unique := uuid.New().String()
	pageId := unique[len(unique)-12:]

	switch r.Method {
	case "POST":
		// Declare a new Person struct.
		var request types.Request

		// Try to decode the request body into the struct. If there is an error,
		// respond to the client with the error message and a 400 status code.
		err := json.NewDecoder(r.Body).Decode(&request)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Println("[ Engine ] Add flow for page #" + pageId)

		rootChannel := make(chan interface{})

		go func(rootChannel chan interface{}) {
			log.Println("[ Engine ] Page #" + pageId + " running flow ")

			if len(request.Flow) > 0 {

				page := engineBrowser.MustPage(request.WebPage).MustWaitLoad()

				/**
				 * If website is HTML only and not rendered with JavaScript
				 * let skip browser to disable download the resources like
				 * image, stylesheet, media, ping, font
				 */
				if request.HtmlOnly != "" {
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

				html := make(map[string]string)

				isFinish := HandleFlowLoop(request.Flow, 0, len(request.Flow), page, html)

				if isFinish {
					page.MustClose()
				}

				resultJson := types.Response{
					Id:   pageId,
					Code: 200,
					Html: html,
				}

				jsonPath := jsonDirectory + pageId + ".json"
				file, _ := json.MarshalIndent(resultJson, "", " ")

				_ = ioutil.WriteFile(jsonPath, file, 0644)

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
	default:
		resultJson := types.Response{
			Code:    403,
			Message: "Method not allowed for this request",
		}

		HandleResponse(w, resultJson, pageId)
	}

	log.Println("[ Engine ] Page #" + pageId + " closed")
}

func HandleFlowLoop(flow []types.Flow, start int, total int, page *rod.Page, html map[string]string) bool {
	if start < total {
		unique := uuid.New().String()
		pageId := unique[len(unique)-12:]

		isFinish := HandleStepLoop(flow[start].Step, 0, len(flow[start].Step), page, pageId, html)

		if isFinish {
			return HandleFlowLoop(flow, start+1, total, page, html)
		}
	}

	if start == total {
		return true
	}

	return false
}

func HandleStepLoop(step []types.Step, start int, total int, page *rod.Page, pageId string, html map[string]string) bool {
	if start == total {
		return true
	} else {
		stepData := step[start]

		var hasElement bool = false
		var detectedElement *rod.Element

		if stepData.Element != "" {
			hasElement = true
		}

		if hasElement {
			_, element, errorMessage := page.Has(stepData.Element)

			if errorMessage != nil {
				panic(errorMessage)
			}

			detectedElement = element
		}

		if stepData.Delay != 0 {
			var sleepTime int = int(stepData.Delay)
			time.Sleep(time.Second * time.Duration(sleepTime))
		} else if stepData.Write != "" {
			detectedElement.MustInput(stepData.Write)
		} else if stepData.Action == "Enter" {
			detectedElement.MustPress(input.Enter)
		} else if stepData.Action == "Click" {
			detectedElement.MustClick()
		} else if stepData.Screenshot.Path != "" {
			screenshotPath := downloadDirectory + pageId + "-" + stepData.Screenshot.Path

			if stepData.Screenshot.Clip.Top != 0 || stepData.Screenshot.Clip.Left != 0 || stepData.Screenshot.Clip.Width != 0 || stepData.Screenshot.Clip.Height != 0 {
				image, _ := page.Screenshot(true, &proto.PageCaptureScreenshot{
					Format:  proto.PageCaptureScreenshotFormatJpeg,
					Quality: gson.Int(100),
					Clip: &proto.PageViewport{
						X:      stepData.Screenshot.Clip.Top,
						Y:      stepData.Screenshot.Clip.Left,
						Width:  stepData.Screenshot.Clip.Width,
						Height: stepData.Screenshot.Clip.Height,
						Scale:  1,
					},
					FromSurface: true,
				})

				_ = utils.OutputFile(screenshotPath, image)
			} else {
				page.MustScreenshot(screenshotPath)
			}
		} else {
			// noop
		}

		return HandleStepLoop(step, start+1, total, page, pageId, html)
	}
}

func HandleResponse(w http.ResponseWriter, data interface{}, pageId string) {
	log.Println("[ Engine ] Page #" + pageId + " sending result")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(data)
}

func Noop(w http.ResponseWriter, r *http.Request) {}
