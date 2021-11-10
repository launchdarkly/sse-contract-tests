package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Filter struct {
	patterns []*regexp.Regexp
}

func (f Filter) String() string {
	var ss []string
	for _, p := range f.patterns {
		ss = append(ss, `"`+p.String()+`"`)
	}
	return strings.Join(ss, " or ")
}

func (f *Filter) Set(value string) error {
	r, err := regexp.Compile(value)
	if err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}
	f.patterns = append(f.patterns, r)
	return nil
}

func (f Filter) IsDefined() bool {
	return len(f.patterns) != 0
}

func (f Filter) AnyMatch(s string) bool {
	for _, p := range f.patterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}
