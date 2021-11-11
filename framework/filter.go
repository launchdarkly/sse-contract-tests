package framework

import (
	"fmt"
	"regexp"
	"strings"
)

// Filter is a function that can determine whether to run a specific test or not.
type Filter func(TestID) bool

type RegexFilters struct {
	MustMatch    RegexList
	MustNotMatch RegexList
}

func (r RegexFilters) AsFilter(id TestID) bool {
	name := id.String()
	return (!r.MustMatch.IsDefined() || r.MustMatch.AnyMatch(name)) &&
		!r.MustNotMatch.AnyMatch(name)
}

type RegexList struct {
	patterns []*regexp.Regexp
}

func (r RegexList) String() string {
	var ss []string
	for _, p := range r.patterns {
		ss = append(ss, `"`+p.String()+`"`)
	}
	return strings.Join(ss, " or ")
}

// Set is called by the command line parser
func (r *RegexList) Set(value string) error {
	rx, err := regexp.Compile(value)
	if err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}
	r.patterns = append(r.patterns, rx)
	return nil
}

func (r RegexList) IsDefined() bool {
	return len(r.patterns) != 0
}

func (r RegexList) AnyMatch(s string) bool {
	for _, p := range r.patterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}

func PrintFilterDescription(harness *TestHarness, filters RegexFilters, allCapabilities []string) {
	if filters.MustMatch.IsDefined() || filters.MustNotMatch.IsDefined() {
		fmt.Println("Some tests will be skipped based on the filter criteria for this test run:")
		if filters.MustMatch.IsDefined() {
			fmt.Printf("  skip any not matching %s\n", filters.MustMatch)
		}
		if filters.MustNotMatch.IsDefined() {
			fmt.Printf("  skip any matching %s\n", filters.MustNotMatch)
		}
		fmt.Println()
	}

	var missingCapabilities []string
	for _, c := range allCapabilities {
		if !harness.TestServiceHasCapability(c) {
			missingCapabilities = append(missingCapabilities, c)
		}
	}
	if len(missingCapabilities) > 0 {
		fmt.Println("Some tests may be skipped because the test service does not support the following capabilities:")
		fmt.Printf("  %s\n", strings.Join(missingCapabilities, ", "))
		fmt.Println()
	}
}
