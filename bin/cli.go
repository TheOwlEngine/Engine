package main

import (
	"bytes"
	"context"
	"encoding/json"
	"engine/types"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DrSmithFr/go-console/pkg/style"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/gosimple/slug"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

var workingDirectory string

func main() {
	io := style.NewConsoleStyler()
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	// enable stylish errors
	defer io.HandleRuntimeException()

	app := &cli.App{
		Name:  "Owl",
		Usage: "This will provide connection to the Engine server",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) error {
			log.Printf("%s Starting OwlEngine v1.0\n", blue("[Owl]"))

			workingDirectory, _ = os.Getwd()
			workingFlows, _ := filepath.Glob(workingDirectory + "/flows/*.yml")

			start := time.Now()
			errorGroup, _ := errgroup.WithContext(context.Background())

			isFinish := sendConfig(workingFlows, 0, len(workingFlows), errorGroup)

			if isFinish {
				end := time.Now()
				fmt.Println("")
				log.Printf("%s All flow finished in %s (s)", blue("[Owl]"), green(end.Sub(start).Seconds()))
			}

			return nil
		},
	}

	err := app.Run(os.Args)

	if err != nil {
		log.Fatal(err)
	}
}

func sendConfig(flows []string, current int, total int, errorGroup *errgroup.Group) bool {
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	start := time.Now()

	if current < total {
		requestChan := make(chan *http.Response, 1)

		fmt.Printf("\n--- Reading flow %s\n\n", green(strings.ReplaceAll(flows[current], workingDirectory+"/flows/", "")))

		config, errorReading := readConfig(flows[current])

		if errorReading != nil {
			log.Fatalf(red("[Owl] Cannot read the config %v"), errorReading)
		}

		if config.Engine == "" {
			log.Fatalf(red("[Owl] Engine server is not specify, you need to specify engine server URL"))
		}

		connectClient := http.Client{
			Timeout: 2 * time.Second,
		}
		_, errorConnection := connectClient.Get(config.Engine)

		if errorConnection != nil {
			log.Fatalf(red("[Owl] Engine server is not running, you need to make sure engine server is reachable"))
		}

		log.Printf("%s Flow %s started", blue("[Owl]"), green(config.Name))
		loading := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		loading.Suffix = "  scraping target " + config.Target
		loading.Start()

		body := types.Config{
			Name:     config.Name,
			Engine:   config.Engine,
			Flow:     config.Flow,
			HtmlOnly: config.HtmlOnly,
			Paginate: config.Paginate,
			Repeat:   config.Repeat,
			Target:   config.Target,
			Record:   config.Record,
		}

		errorGroup.Go(func() error { return sendRequest(body, requestChan) })

		requestResponse := <-requestChan

		defer requestResponse.Body.Close()

		var result types.Response

		resultBody, _ := ioutil.ReadAll(requestResponse.Body)

		loading.Stop()

		json.Unmarshal([]byte(resultBody), &result)

		slugName := slug.Make(result.Name)
		jsonPath := workingDirectory + "/resources/json/" + slugName + "-" + result.Id + ".json"

		log.Printf("%s Result saved : %s", blue("[Owl]"), green(jsonPath))

		_ = ioutil.WriteFile(jsonPath, resultBody, 0644)

		end := time.Now()
		log.Printf("%s Flow #%s finished in %s (s)", blue("[Owl]"), green(result.Id), green(end.Sub(start).Seconds()))
		log.Printf("%s Flow closed", blue("[Owl]"))

		// Delay two second
		time.Sleep(2 * time.Second)

		return sendConfig(flows, current+1, total, errorGroup)
	}

	if current == total {
		return true
	}

	return false
}

func readConfig(filename string) (*types.Config, error) {
	red := color.New(color.FgRed).SprintFunc()

	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &types.Config{}
	err = yaml.Unmarshal(buf, c)

	if err != nil {
		log.Fatalf(red("[Owl] Cannot read flow file %q : %v"), filename, err)
	}

	return c, nil
}

func sendRequest(data types.Config, requestChan chan *http.Response) error {
	body, _ := json.Marshal(data)
	response, httpError := http.Post(data.Engine, "application/json", bytes.NewReader(body))

	requestChan <- response

	return httpError
}
