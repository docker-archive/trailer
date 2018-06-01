package spec

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/onsi/ginkgo/reporters"
)

// TODO: add tests and comments

func ParseFile(file string) ([]reporters.JUnitTestSuite, error) {
	xmlFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	xmlBytes, _ := ioutil.ReadAll(xmlFile)
	suite, err := UnmarshalSingleTestSuite(xmlBytes)
	if err == nil {
		return []reporters.JUnitTestSuite{suite}, nil
	}

	suites, err := UnmarshalMultipleTestSuites(xmlBytes)
	if err == nil {
		return suites, nil
	}
	return nil, fmt.Errorf("failed to parse any testsuites from xml file: %s", file)
}

func UnmarshalSingleTestSuite(xmlBytes []byte) (reporters.JUnitTestSuite, error) {
	var suite reporters.JUnitTestSuite
	xml.Unmarshal(xmlBytes, &suite)

	if len(suite.TestCases) == 0 {
		return reporters.JUnitTestSuite{}, fmt.Errorf("failed to parse single testsuite from xml file")
	}

	return suite, nil
}

func UnmarshalMultipleTestSuites(xmlBytes []byte) ([]reporters.JUnitTestSuite, error) {
	var suites JUnitTestSuites
	xml.Unmarshal(xmlBytes, &suites)

	if len(suites.Suites) == 0 {
		return suites.Suites, fmt.Errorf("failed to parse multiple testsuites from xml file")
	}

	return suites.Suites, nil
}
