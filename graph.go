/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"bufio"
	"bytes"
	"runtime"
	"strings"
	"sync"
)

type Node struct {
	name string
}

func (n *Node) String() string {
	return n.name
}

type edge struct {
	source *Node
	target *Node
}

func (e *edge) String() string {
	return e.source.String() + " => " + e.target.String()
}

// Graph represents the read dependency graph in memory. It satisfies the io.Reader and io.Writer interfaces
type Graph struct {
	nodes []*Node
	edges []*edge
	mutex sync.Mutex
}

func (g *Graph) AddEdge(sourceName string, targetName string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	// add the nodes if they aren't already in place
	var source, target *Node
	for _, nptr := range g.nodes {
		if nptr.name == sourceName {
			source = nptr
		} else if nptr.name == targetName {
			target = nptr
		}
	}
	if source == nil {
		source = &Node{sourceName}
		g.nodes = append(g.nodes, source)
	}
	if target == nil {
		target = &Node{targetName}
		g.nodes = append(g.nodes, target)
	}
	g.edges = append(g.edges, &edge{source: source, target: target})
}

func (g *Graph) Copy() *Graph {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	result := &Graph{
		nodes: make([]*Node, len(g.nodes)),
		edges: make([]*edge, len(g.edges)),
	}
	copy(result.nodes, g.nodes)
	copy(result.edges, g.edges)
	return result
}

func (g *Graph) GetNode(name string) *Node {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, node := range g.nodes {
		if node.name == name {
			return node
		}
	}
	return nil
}

func (g *Graph) GetDependencies(n *Node) []*Node {
	deps := make([]*Node, 0)
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, edge := range g.edges {
		if edge.source == n {
			deps = append(deps, edge.target)
		}
	}
	return deps
}

func (g *Graph) GetDependants(n *Node) []*Node {
	deps := make([]*Node, 0)
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, edge := range g.edges {
		if edge.target == n {
			deps = append(deps, edge.source)
		}
	}
	return deps
}

func (g *Graph) scanDependencies(line string, syntax *Syntax) {
	infixIndex := strings.Index(line, syntax.EdgeInfix)
	sources := []string{line[:infixIndex]}
	if syntax.SourceDelimiter != "" {
		sources = strings.Split(line[:infixIndex], syntax.SourceDelimiter)
	}
	targets := strings.Split(line[infixIndex+len(syntax.EdgeInfix):], syntax.TargetDelimiter)
	for _, source := range sources {
		if syntax.StripWhitespace {
			source = strings.TrimSpace(source)
		}
		if source != "" {
			for _, target := range targets {
				if syntax.StripWhitespace {
					target = strings.TrimSpace(target)
				}
				if target != "" {
					g.AddEdge(source, target)
				}
			}
		}
	}
}

// FromScanner reads data from the given scanner, building up the dependency tree.
func (g *Graph) FromScanner(scanner *bufio.Scanner, syntax ...*Syntax) (*Graph, error) {
	if len(syntax) == 0 {
		panic("FromScanner: At least one syntax required!")
	}
	scanner.Split(scanLineWithEscape)
	activeSyntaxes := make(map[*Syntax]struct{}, len(syntax))
	// for running concurrently, we'll add a pool of worker goroutines
	numWorkers := runtime.GOMAXPROCS(0)
	// we need to wait for our goroutines to finish
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(numWorkers)
	defer waitGroup.Wait()
	type Task struct {
		line   string
		syntax *Syntax
	}
	tasks := make(chan Task, numWorkers)
	defer close(tasks)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer waitGroup.Done()
			for task := range tasks {
				g.scanDependencies(task.line, task.syntax)
			}
		}()
	}
	for scanner.Scan() {
		if scanner.Err() != nil {
			return g, scanner.Err()
		}
		line := scanner.Text()
		for _, syntax := range syntax {
			if strings.Contains(line, syntax.GraphPrefix) {
				activeSyntaxes[syntax] = struct{}{}
			} else if _, active := activeSyntaxes[syntax]; !active {
				continue
			}
			if syntax.GraphSuffix != "" && strings.Contains(line, syntax.GraphSuffix) {
				delete(activeSyntaxes, syntax)
			}
			prefIndex := strings.Index(line, syntax.EdgePrefix)
			infixIndex := strings.Index(line, syntax.EdgeInfix)
			suffixIndex := strings.LastIndex(line, syntax.EdgeSuffix)
			if prefIndex >= 0 && infixIndex >= 0 && suffixIndex >= 0 {
				tasks <- Task{line[prefIndex+len(syntax.EdgePrefix) : suffixIndex], syntax}
				break
			}
		}
	}
	return g, nil
}

func (g *Graph) String() string {
	if len(g.nodes) == 0 {
		return "{empty graph}"
	}
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
		nodes: make([]*Node, 0),
		edges: make([]*edge, 0),
	}
}

func scanLineWithEscape(data []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanLines(data, atEOF)
	for err == nil && len(token) > 0 && token[len(token)-1] == '\\' {
		// omit the trailing backslash
		token = token[:len(token)-1]
		nextData := data[advance:]
		var nextAdvance int
		var nextToken []byte
		nextAdvance, nextToken, err = bufio.ScanLines(nextData, atEOF)
		advance += nextAdvance
		token = append(token, nextToken...)
	}
	return
}
