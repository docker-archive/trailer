package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/educlos/testrail"
	"github.com/urfave/cli"

	"github.com/whoshuu/trailer/spec"
)

type Suite struct {
	ProjectID   int            `yaml:"project_id"`
	SuiteID     int            `yaml:"suite_id"`
	LastUpdated string         `yaml:"last_updated"`
	Cases       map[int]string `yaml:"cases"`
}

func main() {
	username := os.Getenv("TESTRAIL_USERNAME")
	token := os.Getenv("TESTRAIL_TOKEN")

	if username == "" || token == "" {
		log.Fatalf("Need to set TESTRAIL_USERNAME and TESTRAIL_TOKEN")
	}

	var (
		verbose   bool
		dry       bool
		retries   int
		runID     int
		suiteID   int
		projectID int
		comment   string
		file      string
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
		{
			Name:    "download",
			Aliases: []string{"d"},
			Usage:   "Download case specs from TestRail",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "verbose, v",
					Usage:       "turn on debug logs",
					Destination: &verbose,
				},
				cli.IntFlag{
					Name:        "project-id, p",
					Usage:       "TestRail project ID to download cases from",
					Destination: &projectID,
				},
				cli.IntFlag{
					Name:        "suite-id, s",
					Usage:       "TestRail suite ID to download cases from",
					Destination: &suiteID,
				},
				cli.StringFlag{
					Name:        "file, f",
					Usage:       "File to write downloaded cases to",
					Destination: &file,
				},
			},
			Action: func(c *cli.Context) error {
				if projectID == 0 {
					log.Fatalf("Must set --project-id to a non-zero integer")
				}

				if suiteID == 0 {
					log.Fatalf("Must set --suite-id to a non-zero integer")
				}

				client := testrail.NewClient("https://docker.testrail.com", username, token)
				cases, err := client.GetCases(projectID, suiteID)
				if err != nil {
					log.Fatalf("Error getting cases: %s", err)
				}

				s := Suite{
					LastUpdated: time.Unix(0, 0).Format(time.RFC3339Nano),
					ProjectID:   projectID,
					SuiteID:     suiteID,
					Cases:       map[int]string{},
				}

				if file != "" {
					if _, err = os.Stat(file); err == nil {
						data, err := ioutil.ReadFile(file)
						if err != nil {
							log.Fatalf("Error reading file: %s", err)
						}

						err = yaml.Unmarshal(data, &s)
						if err != nil {
							log.Fatalf("Error unmarshaling suite data: %s", err)
						}
					}
				}

				lastUpdated, err := time.Parse(time.RFC3339Nano, s.LastUpdated)
				if err != nil {
					log.Fatalf("Error parsing last_updated time: %s", err)
				}

				updated := false
				for _, c := range cases {
					if lastUpdated.Before(time.Unix(int64(c.UdpatedOn), 0)) {
						s.Cases[c.ID] = c.Title
						updated = true
					}
				}

				if updated {
					s.LastUpdated = time.Now().Format(time.RFC3339Nano)
					data, err := yaml.Marshal(&s)
					if err != nil {
						log.Fatalf("Error marshaling suite data: %s", err)
					}

					if file != "" {
						err = ioutil.WriteFile(file, data, 0644)
						if err != nil {
							log.Fatalf("Error writing suite data to output file: %s", err)
						}
					} else {
						log.Print(string(data))
					}
				}

				return nil
			},
		},
	}

	app.Run(os.Args)
}
