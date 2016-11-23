package main

import (
	"bufio"
	"flag"
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

func main() {
	// declare flags
	syntaxString := flag.String("syntax", "Makefile,Dot", "Syntax to be used to parse the file")
	outfilename := flag.String("out", "", "File to write a dot representation of the dependency tree")
	flag.Parse()
	filenames := flag.Args()
	syntax, err := ParseSyntax(*syntaxString)
	if err != nil {
		panic(err)
	}
	graph, _ := newGraph().FromScanner(bufio.NewScanner(strings.NewReader("s1 s2: dep1 dep2\ns1: dep3\ndep3 dep2: dp4 dep5 dp6\ndep1:dp0\ndep5:dep7")), syntax...)
	println(graph.String())
	print(*outfilename)
	print(len(filenames))
}
