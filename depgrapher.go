/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"bufio"
	"flag"
	"github.com/SimplicityApks/depgrapher/graph"
	"github.com/SimplicityApks/depgrapher/syntax"
	"io"
	"os"
	"strings"
)

func testGraph() {
	g, _ := graph.New().FromScanner(bufio.NewScanner(strings.NewReader("s1 s2: dep1 dep2\ns1: dep3\ndep3 dep2: dp4 dep5 dp6\ndep1:dp0\ndep5:dep7")), syntax.Makefile)
	println("Reading graph successfull")
	println(g.String())
	println("Here is your dependency tree:")
	graph.PrintFullDepTree(g)
	println("Here is the dot export")
	graph.WriteDot(g, os.Stdout)
}

// parseFiles parses the given files with the given syntaxes and returns the generated graphs
func parseFiles(filenames []string, syntax ...*syntax.Syntax) (g graph.Graph, err error) {
	readers := make([]io.Reader, len(filenames))
	for index, filename := range filenames {
		readers[index], err = os.Open(filename)
		if err != nil {
			return nil, err
		}
	}
	scanner := bufio.NewScanner(io.MultiReader(readers...))
	return graph.New().FromScanner(scanner, syntax...)
}

func main() {
	// declare flags
	syntaxString := flag.String("syntax", "Makefile,Dot", "Syntax to be used to parse the file")
	outfilename := flag.String("outfile", "", "File to write a dot representation of the dependency tree")
	flag.Parse()
	filenames := flag.Args()
	s, err := syntax.Parse(*syntaxString)
	if err != nil {
		panic(err)
	}
	var g graph.Graph
	g, err = parseFiles(filenames, s...)
	if err != nil {
		panic(err)
	}
	if *outfilename == "stdout" {
		graph.WriteDot(g, os.Stdout)
	} else if *outfilename != "" {
		var outfile io.Writer
		outfile, err = os.Create(*outfilename)
		graph.WriteDot(g, outfile)
	} else {
		// write ascii graph to stdout
		graph.PrintFullDepTree(g)
	}
}
