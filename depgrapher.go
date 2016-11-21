package main

import (
	"bufio"
	"os"
	"strings"
)

func main() {
	graph, _ := newGraph().FromScanner(bufio.NewScanner(strings.NewReader("s1 s2: dep1 dep2\ns1: dep3\ndep3 dep2: dp4 dep5 dp6\ndep1:dp0\ndep5:dep7")), MakefileSyntax)
	println("Reading graph successfull")
	println(graph.String())
	println("Here is your dependency tree:")
	printFullDepTree(graph)
	println("Here is the dot export")
	writeDot(graph, os.Stdout)
}
