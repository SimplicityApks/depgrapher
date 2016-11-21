// Contains the syntax for reading and writing graphs to files
package main

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

var MakefileSyntax = Syntax{
	GraphPrefix:     "",
	EdgePrefix:      "",
	SourceDelimiter: " ",
	EdgeInfix:       ":",
	TargetDelimiter: " ",
	EdgeSuffix:      "",
	GraphSuffix:     "",
	StripWhitespace: true,
}

var DotSyntax = Syntax{
	GraphPrefix:     "digraph{",
	EdgePrefix:      "",
	SourceDelimiter: "",
	EdgeInfix:       "->",
	TargetDelimiter: "",
	EdgeSuffix:      ";",
	GraphSuffix:     "}",
	StripWhitespace: true,
}
