/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

// Package syntax contains the syntax for reading and writing graphs to files.
// It also comes with predefined variables for common syntaxes.
// The syntax struct assumes that every edge is written in a single line.
package syntax

import (
	"errors"
	"strings"
)

// to be used in Syntax to indicate that the field is not available in the syntax
const IGNOREFIELD = ""

type Syntax struct {
	GraphPrefix     string
	EdgePrefix      string
	SourceDelimiter string
	EdgeInfix       string
	TargetDelimiter string
	EdgeSuffix      string
	GraphSuffix     string
	StripWhitespace bool
}

var Makefile = &Syntax{
	GraphPrefix:     "",
	EdgePrefix:      "",
	SourceDelimiter: " ",
	EdgeInfix:       ":",
	TargetDelimiter: " ",
	EdgeSuffix:      "",
	GraphSuffix:     "",
	StripWhitespace: true,
}

var MakeCall = []*Syntax{
	{
		GraphPrefix:     "",
		EdgePrefix:      "$(call DEPEND_ALL,",
		SourceDelimiter: "",
		EdgeInfix:       ",",
		TargetDelimiter: ",",
		EdgeSuffix:      ")",
		GraphSuffix:     "",
		StripWhitespace: true,
	},
	{
		GraphPrefix:     "",
		EdgePrefix:      "$(call ALL_SPECS,",
		SourceDelimiter: ",",
		EdgeInfix:       "):",
		TargetDelimiter: " ",
		EdgeSuffix:      "",
		GraphSuffix:     "",
		StripWhitespace: true,
	},
}

var Dot = &Syntax{
	GraphPrefix:     "digraph{",
	EdgePrefix:      "",
	SourceDelimiter: "",
	EdgeInfix:       "->",
	TargetDelimiter: "",
	EdgeSuffix:      ";",
	GraphSuffix:     "}",
	StripWhitespace: true,
}

// Parse parses new syntaxes from the given string, supporting various formats.
func Parse(s string) ([]*Syntax, error) {
	result := make([]*Syntax, 0)
	// supported strings (e.g.):
	// Makefile
	// Makefile,Dot
	// Makefile,{"GraphPrefix","EdgePrefix","SourceDelimiter","EdgeInfix","TargetDelimiter","EdgeSuffix","GraphSuffix",true}
	stringList := make([]string, 0)
	for index := strings.IndexAny(s, ",{}\"'"); index > 0; index = strings.IndexAny(s, ",{}\"'") {
		switch s[index] {
		case ',':
			switch token := s[:index]; token {
			case "Makefile", "makefile", "Make", "make", "m":
				result = append(result, Makefile)
			case "MakeCall", "makecall", "c":
				result = append(result, MakeCall...)
			case "Dot", "dot", "d":
				result = append(result, Dot)
			default:
				return result, errors.New("Invalid syntax name: " + token)
			}
		case '{':
			if index != 0 {
				return result, errors.New("Unexpected character(s) before opening bracket: '" + s[:index] + "'")
			}
		case '}':
			if len(stringList) != 6 {
				return result, errors.New("Brackets didn't contain the 7 syntax elements")
			}
			result = append(result, &Syntax{stringList[0], stringList[1], stringList[2], stringList[3], stringList[4], stringList[5], stringList[6], true})
			stringList = make([]string, 0)
		case '"':
			endIndex := strings.Index(s, "\"")
			stringList = append(stringList, s[:endIndex])
			index = endIndex
		case '\'': // TODO
		}
		s = s[index+1:]
	}
	// and parse the last token
	switch s {
	case "Makefile", "makefile", "Make", "make", "m":
		result = append(result, Makefile)
	case "MakeCall", "makecall", "c":
		result = append(result, MakeCall...)
	case "Dot", "dot", "d":
		result = append(result, Dot)
	default:
		return result, errors.New("Invalid syntax name: " + s)
	}
	return result, nil
}
