package main

import (
	"bufio"
	"strings"
	"bytes"
)

type node struct {
	name string
}

func (n *node) String() string {
	return n.name
}

type edge struct {
	source *node
	target *node
}

func (e *edge) String() string {
	return e.source.String() + " => " + e.target.String()
}

// Graph represents the read dependency graph in memory. It satisfies the io.Reader and io.Writer interfaces
type Graph struct {
	nodes []*node
	edges []*edge
}

func (g *Graph) AddEdge(source *node, target *node) {
	// add the nodes if they aren't already in place
	var foundSource, foundTarget bool
	for _, nptr := range g.nodes {
		if nptr == source {
			foundSource = true
		} else if nptr == target {
			foundTarget = true
		}
	}
	if !foundSource {
		g.nodes = append(g.nodes, source)
	}
	if !foundTarget {
		g.nodes = append(g.nodes, target)
	}
	g.edges = append(g.edges, &edge{source: source, target: target})
}


func (g *Graph) scanDependencies(line string, syntax Syntax) {
	//sourceSepIndex, infixIndex := strings.Index(line, syntax.sourceDelimiter), strings.Index(line, syntax.infix)
	infixIndex := strings.Index(line, syntax.infix)
	sources := strings.Split(line[:infixIndex], syntax.sourceDelimiter)
	targets := strings.Split(line[infixIndex+len(syntax.infix):], syntax.targetDelimiter)
	for _, source := range sources {
		if syntax.stripWhitespace {
			source = strings.TrimSpace(source)
		}
		if source != "" {
			for _, target := range targets {
				if syntax.stripWhitespace {
					target = strings.TrimSpace(target)
				}
				if target != "" {
					g.AddEdge(&node{source}, &node{target})
				}
			}
		}
	}
}

// FromScanner reads data from the given scanner, building up the dependency tree.
func (g *Graph) FromScanner(scanner *bufio.Scanner, syntax Syntax) (*Graph, error) {
	scanner.Split(scanLineWithEscape)
	for scanner.Scan() {
		if scanner.Err() != nil {
			return g, scanner.Err()
		}
		line := string(scanner.Text())
		prefIndex := strings.Index(line, syntax.prefix)
		infixIndex := strings.Index(line, syntax.infix)
		suffixIndex := strings.LastIndex(line, syntax.suffix)
		if prefIndex >= 0 && infixIndex >= 0 && suffixIndex >= 0 {
			g.scanDependencies(line[prefIndex+len(syntax.prefix):suffixIndex-1], syntax)
		}
	}
	return g, nil
}

func (g *Graph) String() string {
	var buffer bytes.Buffer
	for _, edge := range g.edges {
		buffer.WriteString(edge.String())
		buffer.WriteString("; ")
	}
	return buffer.String()
}

// newGraph creates a new, empty graph
func newGraph() *Graph {
	return &Graph{
		nodes: make([]*node, 0),
		edges: make([]*edge, 0),
	}
}

type Syntax struct {
	prefix          string
	sourceDelimiter string
	infix           string
	targetDelimiter string
	suffix          string
	stripWhitespace bool
}

func scanLineWithEscape(data []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanLines(data, atEOF)
	for err == nil && len(token)>0 && token[len(token)-1] == '\\' {
		secondAdvance, secondToken, secondErr := bufio.ScanLines(data, atEOF)
		advance += secondAdvance
		token = append(token, secondToken...)
		err = secondErr
	}
	return
}