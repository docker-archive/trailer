package spec

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/educlos/testrail"
	"github.com/onsi/ginkgo/reporters"
)

// TODO: add tests and comments
// TODO: split this up into more pieces

type JUnitTestSuites struct {
	Suites []reporters.JUnitTestSuite `xml:"testsuite"`
}

type TestStatus int

const (
	Passed TestStatus = iota
	Failed
	Skipped
)

type Update struct {
	Status  TestStatus
	Message string
	Elapsed time.Duration
}

type Updates struct {
	ResultMap map[int]Update
}

func (u *Updates) AddSuites(comment string, suites JUnitTestSuites) error {
	for _, suite := range suites.Suites {
		for _, test := range suite.TestCases {
			regex, err := regexp.Compile("TestRailC([\\d]+)")
			if err != nil {
				return fmt.Errorf("failed to compile test case regex: %s", err)
			}
			ids := regex.FindAllStringSubmatch(test.Name, -1)
			for _, id := range ids {
				if len(id) != 2 {
					return fmt.Errorf("failed to parse case ID")
				}
				update := Update{
					Status:  Passed,
					Elapsed: time.Duration(test.Time) * time.Second,
				}
				if test.Skipped != nil {
					update.Status = Skipped
				}
				if test.FailureMessage != nil {
					update.Status = Failed
					update.Message = fmt.Sprintf("%s\n\n%s", comment, (*test.FailureMessage).Message)
				}
				i, err := strconv.Atoi(id[1])
				if err != nil {
					return fmt.Errorf("failed to convert case ID to integer")
				}
				if r, ok := u.ResultMap[i]; ok {
					if r.Status == Failed {
						continue
					}
				}

				u.ResultMap[i] = update
			}
		}
	}

	return nil
}

func (u *Updates) CreatePayload() (testrail.SendableResultsForCase, error) {
	results := testrail.SendableResultsForCase{
		Results: []testrail.ResultsForCase{},
	}

	for k, v := range u.ResultMap {
		result := testrail.SendableResult{
			StatusID: 1,
		}
		timespan := testrail.TimespanFromDuration(v.Elapsed)
		if timespan != nil {
			result.Elapsed = *timespan
		}
		if v.Status == Failed {
			result.StatusID = 5
			result.Comment = v.Message
		}
		if v.Status == Skipped {
			result.StatusID = 3
		} else {
			results.Results = append(results.Results, testrail.ResultsForCase{k, result})
		}
	}

	return results, nil
}

func (u *Updates) RemoveResult(i int) {
	delete(u.ResultMap, i)
}
