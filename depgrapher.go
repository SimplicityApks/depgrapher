package main

import (
	"bufio"
	"flag"
	"io"
	"os"
	"strings"
)

func testGraph() {
	graph, _ := newGraph().FromScanner(bufio.NewScanner(strings.NewReader("s1 s2: dep1 dep2\ns1: dep3\ndep3 dep2: dp4 dep5 dp6\ndep1:dp0\ndep5:dep7")), MakefileSyntax)
	println("Reading graph successfull")
	println(graph.String())
	println("Here is your dependency tree:")
	printFullDepTree(graph)
	println("Here is the dot export")
	writeDot(graph, os.Stdout)
}

// parseFiles parses the given files with the given syntaxes and returns the generated graphs
func parseFiles(filenames []string, syntax ...*Syntax) (g *Graph, err error) {
	readers := make([]io.Reader, len(filenames))
	for index, filename := range filenames {
		readers[index], err = os.Open(filename)
		if err != nil {
			return nil, err
		}
	}
	scanner := bufio.NewScanner(io.MultiReader(readers...))
	return newGraph().FromScanner(scanner, syntax...)
}

func main() {
	// declare flags
	syntaxString := flag.String("syntax", "Makefile,Dot", "Syntax to be used to parse the file")
	outfilename := flag.String("outfile", "", "File to write a dot representation of the dependency tree")
	flag.Parse()
	filenames := flag.Args()
	syntax, err := ParseSyntax(*syntaxString)
	if err != nil {
		panic(err)
	}
	var graph *Graph
	graph, err = parseFiles(filenames, syntax...)
	if err != nil {
		panic(err)
	}
	if *outfilename == "stdout" {
		writeDot(graph, os.Stdout)
	} else if *outfilename != "" {
		var outfile io.Writer
		outfile, err = os.Create(*outfilename)
		writeDot(graph, outfile)
	} else {
		// write ascii graph to stdout
		printFullDepTree(graph)
	}
}
