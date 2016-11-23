// Contains the syntax for reading and writing graphs to files
package main

import (
	"errors"
	"strings"
)

// to be used in Syntax to indicate that the field is not available in the syntax
const IGNOREFIELD = ""

// assumes that every edge is written in a single line
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

var MakefileSyntax = &Syntax{
	GraphPrefix:     "",
	EdgePrefix:      "",
	SourceDelimiter: " ",
	EdgeInfix:       ":",
	TargetDelimiter: " ",
	EdgeSuffix:      "",
	GraphSuffix:     "",
	StripWhitespace: true,
}

var DotSyntax = &Syntax{
	GraphPrefix:     "digraph{",
	EdgePrefix:      "",
	SourceDelimiter: "",
	EdgeInfix:       "->",
	TargetDelimiter: "",
	EdgeSuffix:      ";",
	GraphSuffix:     "}",
	StripWhitespace: true,
}

func ParseSyntax(s string) ([]*Syntax, error) {
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
				result = append(result, MakefileSyntax)
			case "Dot", "dot", "d":
				result = append(result, DotSyntax)
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
		result = append(result, MakefileSyntax)
	case "Dot", "dot", "d":
		result = append(result, DotSyntax)
	default:
		return result, errors.New("Invalid syntax name: " + s)
	}
	return result, nil
}
