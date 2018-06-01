package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/educlos/testrail"
	"github.com/urfave/cli"

	"github.com/docker/trailer/spec"
)

func main() {
	username := os.Getenv("TESTRAIL_USERNAME")
	token := os.Getenv("TESTRAIL_TOKEN")

	if username == "" || token == "" {
		log.Fatalf("Need to set TESTRAIL_USERNAME and TESTRAIL_TOKEN")
	}

	var (
		verbose bool
		dry     bool
		retries int
		runID   int
		comment string
	)

	app := cli.NewApp()
	app.HideHelp = true
	app.HideVersion = true
	app.Name = "trailer"
	app.Usage = "TestRail command line utility"
	app.Commands = []cli.Command{
		{
			Name:    "upload",
			Aliases: []string{"u"},
			Usage:   "Upload JUnit XML reports to TestRail",
			Flags: []cli.Flag{
				// TODO: Respect verbosity and use a proper logging library
				cli.BoolFlag{
					Name:        "verbose, v",
					Usage:       "turn on debug logs",
					Destination: &verbose,
				},
				cli.BoolFlag{
					Name:        "dry, d",
					Usage:       "print readable results without updating TestRail run",
					Destination: &dry,
				},
				cli.IntFlag{
					Name:        "ignore-failures, i",
					Usage:       "ignore failures and retry this number of times",
					Destination: &retries,
					Value:       1,
				},
				cli.IntFlag{
					Name:        "run-id, r",
					Usage:       "TestRail run ID to target for the update",
					Destination: &runID,
				},
				cli.StringFlag{
					Name:        "comment, c",
					Usage:       "prefix to use when commenting on TestRail updates",
					Destination: &comment,
				},
			},
			ArgsUsage: "[input *.xml files...]",
			Action: func(c *cli.Context) error {
				if runID == 0 {
					log.Fatalf("Must set --run-id to a non-zero integer")
				}

				updates := spec.Updates{
					ResultMap: map[int]spec.Update{},
				}

				suites := spec.JUnitTestSuites{}
				for _, file := range c.Args() {
					newSuites, err := spec.ParseFile(file)
					if err != nil {
						log.Fatalf(fmt.Sprintf("Failed to parse file: %s", err))
					}

					suites.Suites = append(suites.Suites, newSuites...)
				}

				updates.AddSuites(comment, suites)

				if !dry {
					client := testrail.NewClient("https://docker.testrail.com", username, token)
					for i := 0; i < retries; i++ {
						results, err := updates.CreatePayload()
						if err != nil {
							log.Fatalf(fmt.Sprintf("Failed to create results payload: %s", err))
						}
						r, err := client.AddResultsForCases(runID, results)
						if err != nil {
							errString := err.Error()
							if strings.Contains(errString, "400 Bad Request") {
								regex, err := regexp.Compile("case C([\\d]+) unknown")
								if err != nil {
									log.Fatalf("failed to compile test case regex: %s", err)
								}
								ids := regex.FindAllStringSubmatch(errString, -1)
								for _, id := range ids {
									if len(id) != 2 {
										log.Fatalf("failed to parse case ID")
									}
									caseID, err := strconv.Atoi(id[1])
									if err != nil {
										log.Fatalf("failed to convert case ID to integer: %s", err)
									}
									updates.RemoveResult(caseID)
								}
							} else {
								log.Fatalf(fmt.Sprintf("Failed to upload test results to TestRail: %s", err))
							}
						}

						if len(r) == 0 {
							log.Print("No results uploaded")
						} else {
							for _, res := range r {
								fmt.Printf("%+v\n", res)
							}
							break
						}
					}
				}

				return nil
			},
		},
	}

	app.Run(os.Args)
}
