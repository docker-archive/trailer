package spec

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalSingleTestSuite(t *testing.T) {
	testcases := []struct{
		payload []byte
		shouldError bool
		numTestCases int
	}{
		{
			payload: []byte(`<?xml version="1.0" encoding="UTF-8"?>
  <testsuite name="Test Suite" tests="4" failures="0" errors="0" time="795">
      <testcase name="testcase1" classname="class1" time="10.193520112"></testcase>
      <testcase name="testcase2" classname="class1" time="0.234914942"></testcase>
      <testcase name="testcase3" classname="class1" time="0.233229868"></testcase>
      <testcase name="testcase4" classname="class1" time="0.221499186"></testcase>
  </testsuite>`),
			shouldError: false,
			numTestCases: 4,
		},
		{
			payload: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="Mocha Tests" time="15.358" tests="4" failures="0">
  <testsuite name="Root Suite" timestamp="2019-09-06T00:23:17" tests="0" failures="0" time="0">
  </testsuite>
  <testsuite name="Test Suite" tests="4" failures="0" errors="0" time="795">
      <testcase name="testcase1" classname="class1" time="10.193520112"></testcase>
      <testcase name="testcase2" classname="class1" time="0.234914942"></testcase>
      <testcase name="testcase3" classname="class1" time="0.233229868"></testcase>
      <testcase name="testcase4" classname="class1" time="0.221499186"></testcase>
  </testsuite>`),
			shouldError: true,
			numTestCases: 0,
		},
	}

	for _, testcase := range testcases {
		result, err := UnmarshalSingleTestSuite(testcase.payload)
		if err != nil {
			assert.True(t, testcase.shouldError)
		} else {
			assert.False(t, testcase.shouldError)
			assert.Equal(t, testcase.numTestCases, len(result.TestCases))
		}
	}
}

func TestUnmarshalMultipleTestSuites(t *testing.T) {
	testcases := []struct{
		payload []byte
		shouldError bool
		numTestSuites int
		numTestCases int
	}{
		{
			payload: []byte(`<?xml version="1.0" encoding="UTF-8"?>
  <testsuite name="Test Suite" tests="4" failures="0" errors="0" time="795">
      <testcase name="testcase1" classname="class1" time="10.193520112"></testcase>
      <testcase name="testcase2" classname="class1" time="0.234914942"></testcase>
      <testcase name="testcase3" classname="class1" time="0.233229868"></testcase>
      <testcase name="testcase4" classname="class1" time="0.221499186"></testcase>
  </testsuite>`),
			shouldError: true,
			numTestSuites: 0,
			numTestCases: 0,
		},
		{
			payload: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="Mocha Tests" time="15.358" tests="4" failures="0">
  <testsuite name="Root Suite" timestamp="2019-09-06T00:23:17" tests="0" failures="0" time="0">
      <testcase name="testcase1" classname="class1" time="10.193520112"></testcase>
  </testsuite>
  <testsuite name="Test Suite" tests="4" failures="0" errors="0" time="795">
      <testcase name="testcase2" classname="class1" time="10.193520112"></testcase>
      <testcase name="testcase3" classname="class1" time="0.234914942"></testcase>
      <testcase name="testcase4" classname="class1" time="0.233229868"></testcase>
      <testcase name="testcase5" classname="class1" time="0.221499186"></testcase>
  </testsuite>
</testsuites>`),
			shouldError: false,
			numTestSuites: 2,
			numTestCases: 5,
		},
	}

	for _, testcase := range testcases {
		result, err := UnmarshalMultipleTestSuites(testcase.payload)
		if err != nil {
			assert.True(t, testcase.shouldError)
		} else {
			assert.False(t, testcase.shouldError)
			assert.Equal(t, testcase.numTestSuites, len(result))
			var totalCount int
			for _, testsuite := range result {
				totalCount += len(testsuite.TestCases)
			}
			assert.Equal(t, testcase.numTestCases, totalCount)
		}
	}
}
