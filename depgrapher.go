package main

import (
	"bufio"
	"strings"
)

var makesyntax = Syntax{
	prefix:          "",
	sourceDelimiter: " ",
	infix:           ":",
	targetDelimiter: " ",
	suffix:          "",
	stripWhitespace: true,
}

func main() {
	graph, _ := newGraph().FromScanner(bufio.NewScanner(strings.NewReader("s1 s2: dep1 dep2\n")), makesyntax)
	println("Reading graph successfull")
	println(graph.String())

}
