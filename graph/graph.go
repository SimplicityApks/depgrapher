/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

// Package graph contains the graph data structure to represent dependency graphs.
// Nodes are fmt.Stringers, and are uniquely identified by their String() method.
// Edges don't have a public interface, but the various graph.*Edge methods work with them.
package graph

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/SimplicityApks/depgrapher/syntax"
	"runtime"
	"strings"
	"sync"
)

// Node represents a single data point stored in a Graph. Its String() method should return a unique string identifier.
// The String() method call should be fast (ideally inline), as it will be called often!
type Node fmt.Stringer

// node is a simple string type, created only by the graph input methods.
type node string

func (s node) String() string {
	return string(s)
}

type edge struct {
	source Node
	target Node
}

func (e *edge) String() string {
	return e.source.String() + " => " + e.target.String()
}

// Graph is the interface for a graph-like data structure.
type Graph interface {
	// GetNode returns the Node with the specified name, or nil if the name couldn't be found in the Graph.
	GetNode(name string) Node
	// GetNodes returns all Node objects saved in the Graph as a slice. If the Graph is empty, an empty slice is returned.
	GetNodes() []Node

	// AddEdge adds a new edge from source to target to the Graph.
	AddEdge(source, target Node)
	// HasEdge returns true if the Graph has an edge from the source Node to the target Node, false otherwise.
	HasEdge(source, target string) bool
	// GetDependencies returns a slice containing all dependencies of the Node with the given string.
	GetDependencies(node string) []Node
	// GetDependencies returns a slice containing all dependants of the Node with the given string.
	GetDependants(node string) []Node

	// Copy returns a deep copy of the Graph.
	Copy() Graph
}

// graph represents the read dependency graph in memory. It satisfies the Graph and fmt.Stringer interfaces
type graph struct {
	nodes []Node
	edges []edge
	mutex sync.Mutex
}

// GetNode returns the Node with the specified name, or nil if the name couldn't be found in g.
func (g *graph) GetNode(name string) Node {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, node := range g.nodes {
		if node.String() == name {
			return node
		}
	}
	return nil
}

func (g *graph) GetNodes() []Node {
	return g.nodes
}

// AddEdge adds a new edge from source to target to the graph.
func (g *graph) AddEdge(source, target Node) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	// add the nodes if they aren't already in place
	var foundSource, foundTarget bool
	for _, n := range g.nodes {
		if n.String() == source.String() {
			foundSource = true
		} else if n.String() == target.String() {
			foundTarget = true
		}
	}
	if !foundSource {
		g.nodes = append(g.nodes, source)
	}
	if !foundTarget {
		g.nodes = append(g.nodes, target)
	}
	g.edges = append(g.edges, edge{source: source, target: target})
}

// HasEdge returns true if g has an edge from the source Node to the target Node, false otherwise.
func (g *graph) HasEdge(source, target string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, edge := range g.edges {
		if edge.source.String() == source && edge.target.String() == target {
			return true
		}
	}
	return false
}

// GetDependencies returns a slice containing all dependencies of the Node with the given string.
// g contains an edge from the Node to each item in the returned dependencies.
func (g *graph) GetDependencies(node string) []Node {
	deps := make([]Node, 0)
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, edge := range g.edges {
		if edge.source.String() == node {
			deps = append(deps, edge.target)
		}
	}
	return deps
}

// GetDependencies returns a slice containing all dependants of the Node with the given string.
// g contains an edge from each item in dependants to the Node.
func (g *graph) GetDependants(node string) []Node {
	deps := make([]Node, 0)
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, edge := range g.edges {
		if edge.target.String() == node {
			deps = append(deps, edge.source)
		}
	}
	return deps
}

// Copy returns a deep copy of g.
func (g *graph) Copy() Graph {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	result := &graph{
		nodes: make([]Node, len(g.nodes)),
		edges: make([]edge, len(g.edges)),
	}
	copy(result.nodes, g.nodes)
	copy(result.edges, g.edges)
	return result
}

// scanDependencies adds the given dependency line with the given syntax as edges to g.
func (g *graph) scanDependencies(line string, syntax *syntax.Syntax) {
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
					g.AddEdge(node(source), node(target))
				}
			}
		}
	}
}

// FromScanner reads data from the given scanner, building up the dependency tree.
func (g *graph) FromScanner(scanner *bufio.Scanner, syntaxes ...*syntax.Syntax) (*graph, error) {
	if len(syntaxes) == 0 {
		panic("FromScanner: At least one syntax required!")
	}
	scanner.Split(scanLineWithEscape)
	activeSyntaxes := make(map[*syntax.Syntax]struct{}, len(syntaxes))
	// for running concurrently, we'll add a pool of worker goroutines
	numWorkers := runtime.GOMAXPROCS(0)
	// we need to wait for our goroutines to finish
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(numWorkers)
	defer waitGroup.Wait()
	type Task struct {
		line   string
		syntax *syntax.Syntax
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
		for _, syntax := range syntaxes {
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

func (g *graph) String() string {
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

// New creates a new, empty graph
func New() *graph {
	return &graph{
		nodes: make([]Node, 0),
		edges: make([]edge, 0),
	}
}

// scanLineWithEscape is a drop-in replacement for bufio.ScanLines, appending the next line if the last byte is a backslash '\'
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
