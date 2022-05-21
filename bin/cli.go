package main

import (
	"bytes"
	"encoding/json"
	"engine/types"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/DrSmithFr/go-console/pkg/style"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v3"
)

func main() {
	io := style.NewConsoleStyler()

	// enable stylish errors
	defer io.HandleRuntimeException()

	app := &cli.App{
		Name:  "Owl",
		Usage: "This will provide connection to the Engine server",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) error {
			workingDirectory, _ := os.Getwd()
			workingFlows, _ := filepath.Glob(workingDirectory + "/flows/*.yml")

			var wg sync.WaitGroup

			wg.Add(len(workingFlows))

			log.Println("[Owl] Start reading flow config")

			for flowIndex, flow := range workingFlows {
				flowData := flow

				go func(flowIndex int) {
					config, errorReading := readConfig(flowData)

					if errorReading != nil {
						log.Fatal(errorReading)
					}

					if config.Engine == "" {
						log.Fatal("[Owl] Engine server is not specify, you need to specify engine server URL")
					}

					connectClient := http.Client{
						Timeout: 2 * time.Second,
					}
					_, errorConnection := connectClient.Get(config.Engine)

					if errorConnection != nil {
						log.Fatal("[Owl] Engine server is not running, you need to make sure engine server is reachable")
					}

					log.Println("[Owl] Sending " + flowData + " config to the server")
					request := types.Request{
						Engine: config.Engine,
						Target: config.Target,
						Flow:   config.Flow,
					}

					sendRequest(request)

					defer wg.Done()
				}(flowIndex)
			}

			wg.Wait()

			log.Println("[Owl] All flow already sended")
			log.Println("[Owl] Application closed")

			return nil
		},
	}

	err := app.Run(os.Args)

	if err != nil {
		log.Fatal(err)
	}
}

func readConfig(filename string) (*types.Config, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &types.Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("[Owl] Cannot read flow file %q: %v", filename, err)
	}

	return c, nil
}

func sendRequest(data types.Request) types.Response {
	var result types.Response

	body, _ := json.Marshal(data)
	response, _ := http.Post(data.Engine, "application/json", bytes.NewBuffer(body))
	resultBody, _ := ioutil.ReadAll(response.Body)

	json.Unmarshal([]byte(resultBody), &result)

	return result
}
