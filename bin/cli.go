package main

import (
	"bytes"
	"context"
	"encoding/json"
	"engine/lib"
	"engine/types"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

var directory string

/**
 * Connector v1.0.0
 *
 * Provide command line function to run all flows and send it into Engine server.
 */
func main() {
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	app := &cli.App{
		Name:  "Owl",
		Usage: "This will provide connection to the Engine server",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) error {
			log.Printf("%s Starting Connector v1.0\n", blue("[OWL]"))

			directory, _ = os.Getwd()
			flows, _ := filepath.Glob(directory + "/flows/*.yml")

			start := time.Now()
			errorGroup, _ := errgroup.WithContext(context.Background())

			async := lib.Async(func() interface{} {
				return request(flows, 0, len(flows), errorGroup)
			})

			isFinish := fmt.Sprintf("%v", async.Await())

			if isFinish == "true" {
				end := time.Now()
				println("")
				log.Printf("%s All flow finished in %s (s)", blue("[OWL]"), green(end.Sub(start).Seconds()))
			}

			return nil
		},
	}

	err := app.Run(os.Args)

	if err != nil {
		log.Fatal(err)
	}
}

/**
 * Function to send flow into Engine using HTTP Request (POST) and format JSON
 */
func request(flows []string, current int, total int, errorGroup *errgroup.Group) bool {
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	start := time.Now()

	if current < total {
		requestChan := make(chan *http.Response, 1)

		fmt.Printf("\n--- Reading flow %s\n\n", green(strings.ReplaceAll(flows[current], directory+"/flows/", "")))

		config, errorReading := config(flows[current])

		if errorReading != nil {
			log.Fatalf(red("[OWL] Cannot read the config %v"), errorReading)
		}

		if config.Engine == "" {
			log.Fatalf(red("[OWL] Engine server is not specify, you need to specify engine server URL"))
		}

		connectClient := http.Client{
			Timeout: 2 * time.Second,
		}
		_, errorConnection := connectClient.Get(config.Engine)

		if errorConnection != nil {
			log.Fatalf(red("[OWL] Engine %s is not running, you need to make sure engine server is reachable"), config.Engine)
		}

		log.Printf("%s Flow %s started", blue("[OWL]"), green(config.Name))
		loading := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		loading.Suffix = "  scraping website " + config.FirstPage
		loading.Start()

		body := types.Config{
			Name:           config.Name,
			Engine:         config.Engine,
			FirstPage:      config.FirstPage,
			ItemsOnPage:    config.ItemsOnPage,
			Infinite:       config.Infinite,
			InfiniteScroll: config.InfiniteScroll,
			Paginate:       config.Paginate,
			PaginateButton: config.PaginateButton,
			PaginateLimit:  config.PaginateLimit,
			Proxy:          config.Proxy,
			ProxyCountry:   config.ProxyCountry,
			Record:         config.Record,
			Flow:           config.Flow,
		}

		errorGroup.Go(func() error { return client(body, requestChan) })

		requestResult := <-requestChan

		defer requestResult.Body.Close()

		var result types.Result

		resultBody, _ := ioutil.ReadAll(requestResult.Body)

		loading.Stop()

		json.Unmarshal([]byte(resultBody), &result)

		jsonPath := directory + "/resources/json/" + result.Slug + ".json"

		log.Printf("%s Result saved : %s", blue("[OWL]"), green(jsonPath))

		_ = ioutil.WriteFile(jsonPath, resultBody, 0644)

		end := time.Now()
		log.Printf("%s Flow #%s finished in %s (s)", blue("[OWL]"), green(result.Id), green(end.Sub(start).Seconds()))
		log.Printf("%s Flow closed", blue("[OWL]"))

		// Delay two second
		time.Sleep(2 * time.Second)

		return request(flows, current+1, total, errorGroup)
	}

	if current == total {
		return true
	}

	return false
}

/**
 * Function to parse YAML file into config struct
 */
func config(filename string) (*types.Config, error) {
	red := color.New(color.FgRed).SprintFunc()

	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &types.Config{}
	err = yaml.Unmarshal(buf, c)

	if err != nil {
		log.Fatalf(red("[OWL] Cannot read flow file %q : %v"), filename, err)
	}

	return c, nil
}

/**
 * HTTP Client function based on http go library
 */
func client(data types.Config, requestChan chan *http.Response) error {
	body, _ := json.Marshal(data)
	result, httpError := http.Post(data.Engine, "application/json", bytes.NewReader(body))

	requestChan <- result

	return httpError
}
