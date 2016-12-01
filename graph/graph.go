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
	"errors"
	"fmt"
	"github.com/SimplicityApks/depgrapher/syntax"
	"runtime"
	"strings"
	"sync"
)

// Node represents a single data point stored in a Graph. Its String() method should return a unique string identifier.
// The String() method call should be fast (ideally inline), as it will be called often!
type Node fmt.Stringer

// node is a simple string type, created only by the Graph input methods.
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

// Interface defines a basic graph-like data structure.
// The stored data is accessed by the String() method, so every Node needs to implement fmt.Stringer.
type Interface interface {
	// AddNode adds the node with edges to the nodes with the given target names to this graph.
	AddNode(node Node, targetNames ...string)
	// GetNode returns the Node with the specified name, or nil if the name couldn't be found in the graph.
	GetNode(name string) Node
	// GetNodes returns all Node objects saved in the graph as a slice. If the graph is empty, an empty slice is returned.
	GetNodes() []Node
	// RemoveNode removes the Node with the given name including its edges from the graph.
	// Returns false if the graph didn't have a matching Node, true otherwise.
	RemoveNode(name string) bool

	// AddEdge adds a new edge from source to target to the graph. The nodes are added to the graph if they were not present.
	AddEdge(source, target Node)
	// HasEdge returns true if the graph has an edge from the source Node to the target Node, false otherwise.
	HasEdge(source, target string) bool
	// RemoveEdge removes the edge from the source Node to the target Node.
	// Returns false if the graph didn't have an edge from source to target, true otherwise.
	RemoveEdge(source, target string) bool
	// GetDependencies returns a slice containing all dependencies of the Node with the given string.
	GetDependencies(node string) []Node
	// GetDependencies returns a slice containing all dependants of the Node with the given string.
	GetDependants(node string) []Node

	// Copy returns a deep copy of the graph.
	Copy() Interface
}

// Graph represents the dependency graph in memory. It internally uses a mutex to make read and write access concurrency-safe
// It satisfies the graph.Interface and fmt.Stringer interfaces.
// The zero value is an empty graph with no dependencies.
//
// This implementation prefers fast(-ish) node lookups over edge removals because it keeps a list of all nodes ready.
type Graph struct {
	nodes []Node
	edges []edge
	mutex sync.Mutex
}

// AddNode adds the node with edges to the nodes with the given target names to this graph.
func (g *Graph) AddNode(node Node, targetNames ...string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.nodes = append(g.nodes, node)
	for _, name := range targetNames {
		var target Node
		for _, n := range g.nodes {
			if n.String() == name {
				target = n
				break
			}
		}
		if target == nil {
			panic(errors.New("AddNode: target node with name " + name + " not present in Graph"))
		}
		g.edges = append(g.edges, edge{source: node, target: target})
	}
}

// GetNode returns the Node with the specified name, or nil if the name couldn't be found in g.
func (g *Graph) GetNode(name string) Node {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, node := range g.nodes {
		if node.String() == name {
			return node
		}
	}
	return nil
}

// GetNodes returns all Node objects saved in the Graph as a slice. If it is empty, an empty slice is returned.
func (g *Graph) GetNodes() []Node {
	return g.nodes
}

// RemoveNode removes the Node with the given name including its edges from the graph.
// Returns false if the graph didn't have a matching Node, true otherwise.
func (g *Graph) RemoveNode(name string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	var node Node
	for i, n := range g.nodes {
		if n.String() == name {
			// safe delete without preserving order
			g.nodes[i] = g.nodes[len(g.nodes)-1]
			g.nodes[len(g.nodes)-1] = nil
			g.nodes = g.nodes[:len(g.nodes)-1]
			node = n
			break
		}
	}
	if node == nil {
		return false
	}
	for i, e := range g.edges {
		if e.source == node || e.target == node {
			// safe delete without preserving order
			g.edges[i] = g.edges[len(g.edges)-1]
			g.edges[len(g.edges)-1] = edge{}
			g.edges = g.edges[:len(g.edges)-1]
		}
	}
	return true
}

// AddEdge adds a new edge from source to target to the Graph. The nodes are added to the Graph if they were not present.
func (g *Graph) AddEdge(source, target Node) {
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
func (g *Graph) HasEdge(source, target string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, edge := range g.edges {
		if edge.source.String() == source && edge.target.String() == target {
			return true
		}
	}
	return false
}

// RemoveEdge removes the edge from the source Node to the target Node.
// Returns false if the graph didn't have an edge from source to target, true otherwise.
func (g *Graph) RemoveEdge(source, target string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for index, e := range g.edges {
		if e.source.String() == source && e.target.String() == target {
			// delete without preserving order as it is faster
			last := len(g.edges) - 1
			g.edges[index] = g.edges[last]
			// set the last edge to nil so the nodes can be garbage collected
			g.edges[last] = edge{}
			g.edges = g.edges[:last]
			return true
		}
	}
	return false
}

// GetDependencies returns a slice containing all dependencies of the Node with the given string.
// The Graph contains an edge from the Node to each item in the returned dependencies.
func (g *Graph) GetDependencies(node string) []Node {
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
// The Graph contains an edge from each item in dependants to the Node.
func (g *Graph) GetDependants(node string) []Node {
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

// Copy returns a deep copy of the Graph.
func (g *Graph) Copy() Interface {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	result := &Graph{
		nodes: make([]Node, len(g.nodes)),
		edges: make([]edge, len(g.edges)),
	}
	copy(result.nodes, g.nodes)
	copy(result.edges, g.edges)
	return result
}

// scanDependencies adds the given dependency line with the given syntax as edges to g.
func (g *Graph) scanDependencies(line string, syntax *syntax.Syntax) {
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
func (g *Graph) FromScanner(scanner *bufio.Scanner, syntaxes ...*syntax.Syntax) (*Graph, error) {
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

// String returns a simple string representation consisting of all edges.
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
