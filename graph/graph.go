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
	source string
	target string
}

func (e *edge) String() string {
	return e.source + " => " + e.target
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

	// AddEdge adds an edge from the source Node to the target Node to the graph.
	AddEdge(source, target string)
	// HasEdge returns true if the graph has an edge from the source Node to the target Node, false otherwise.
	HasEdge(source, target string) bool
	// RemoveEdge removes the edge from the source Node to the target Node.
	// Returns false if the graph didn't have an edge from source to target, true otherwise.
	RemoveEdge(source, target string) bool
	// GetDependencies returns a slice containing all dependencies of the Node with the given string.
	GetDependencies(node string) []Node
	// GetDependants returns a slice containing all dependants of the Node with the given string.
	GetDependants(node string) []Node

	// Copy returns a deep copy of the graph.
	Copy() Interface
}

// Graph represents the dependency graph in memory. Reades and writes at the same time are not concurrency-safe and
// therefore need to be synchronized. Checkout graph.Synced for a thread-safe version of Graph.
// It satisfies the graph.Interface and fmt.Stringer interfaces.
// The zero value is an uninitialized graph, use graph.New() to get an initialized Graph.
//
// This implementation prefers fast(-ish) node lookups over edge removals because it keeps a map of all nodes ready.
type Graph struct {
	nodes map[string]Node
	edges []edge
}

// New returns a freshly initialized, ready-to-use Graph.
// It takes an optional first parameter specifying the capacity of nodes that the Graph will contain, and an optional
// second uint specifying the number of edges it will hold.
func New(capacities ...uint) *Graph {
	switch len(capacities) {
	case 0:
		return &Graph{nodes: make(map[string]Node)}
	case 1:
		return &Graph{nodes: make(map[string]Node, capacities[0])}
	case 2:
		return &Graph{nodes: make(map[string]Node, capacities[0]), edges: make([]edge, 0, capacities[1])}
	default:
		panic("More than 2 capacity parameters for graph.New")
	}
}

// AddNode adds the node with edges to the nodes with the given target names to this graph.
func (g *Graph) AddNode(node Node, targetNames ...string) {
	g.nodes[node.String()] = node
	for _, name := range targetNames {
		if _, ok := g.nodes[name]; !ok {
			panic(errors.New("AddNode: target node with name " + name + " not present in Graph"))
		}
		g.edges = append(g.edges, edge{source: node.String(), target: name})
	}
}

// AddNodes adds the given nodes to this graph.
func (g *Graph) AddNodes(nodes ...Node) {
	for _, node := range nodes {
		g.nodes[node.String()] = node
	}
}

// GetNode returns the Node with the specified name, or nil if the name couldn't be found in g.
func (g *Graph) GetNode(name string) Node {
	return g.nodes[name]
}

// GetNodes returns all Node objects saved in the Graph as a slice. If it is empty, an empty slice is returned.
func (g *Graph) GetNodes() []Node {
	nodes := make([]Node, 0, len(g.nodes))
	for _, node := range g.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// RemoveNode removes the Node with the given name including its edges from the graph.
// Returns false if the graph didn't have a matching Node, true otherwise.
func (g *Graph) RemoveNode(name string) bool {
	if _, ok := g.nodes[name]; !ok {
		return false
	}
	delete(g.nodes, name)
	for i, e := range g.edges {
		if e.source == name || e.target == name {
			// safe delete without preserving order
			g.edges[i] = g.edges[len(g.edges)-1]
			g.edges[len(g.edges)-1] = edge{}
			g.edges = g.edges[:len(g.edges)-1]
		}
	}
	return true
}

// AddEdge adds an edge from the source Node to the target Node to the graph.
func (g *Graph) AddEdge(source, target string) {
	if _, ok := g.nodes[source]; !ok {
		panic("AddEdge: source Node not present in Graph!")
	}
	if _, ok := g.nodes[target]; !ok {
		panic("AddEdge: source Node not present in Graph!")
	}
	g.edges = append(g.edges, edge{source: source, target: target})
}

// AddEdgeAndNodes adds a new edge from source to target to the Graph. The nodes are added to the Graph if they were not present.
func (g *Graph) AddEdgeAndNodes(source, target Node) {
	sourceName, targetName := source.String(), target.String()
	// add the nodes if they aren't already in place
	if _, ok := g.nodes[sourceName]; !ok {
		g.nodes[sourceName] = source
	}
	if _, ok := g.nodes[targetName]; !ok {
		g.nodes[targetName] = target
	}
	g.edges = append(g.edges, edge{source: sourceName, target: targetName})
}

// HasEdge returns true if g has an edge from the source Node to the target Node, false otherwise.
func (g *Graph) HasEdge(source, target string) bool {
	for _, edge := range g.edges {
		if edge.source == source && edge.target == target {
			return true
		}
	}
	return false
}

// RemoveEdge removes the edge from the source Node to the target Node.
// Returns false if the graph didn't have an edge from source to target, true otherwise.
func (g *Graph) RemoveEdge(source, target string) bool {
	for index, e := range g.edges {
		if e.source == source && e.target == target {
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
	var deps []Node
	for _, edge := range g.edges {
		if edge.source == node {
			deps = append(deps, g.nodes[edge.target])
		}
	}
	return deps
}

// GetDependants returns a slice containing all dependants of the Node with the given string.
// The Graph contains an edge from each item in dependants to the Node.
func (g *Graph) GetDependants(node string) []Node {
	var deps []Node
	for _, edge := range g.edges {
		if edge.target == node {
			deps = append(deps, g.nodes[edge.source])
		}
	}
	return deps
}

// Copy returns a deep copy of the Graph.
func (g *Graph) Copy() Interface {
	result := &Graph{
		nodes: make(map[string]Node, len(g.nodes)),
		edges: make([]edge, len(g.edges)),
	}
	for k, v := range g.nodes {
		result.nodes[k] = v
	}
	copy(result.edges, g.edges)
	return result
}

// GetDependencyGraph builds the dependency graph for the node. Returns nil if no node with the given name was found in g.
func (g *Graph) GetDependencyGraph(nodename string) *Graph {
	start, ok := g.nodes[nodename]
	if !ok {
		return nil
	}
	var startEdges []edge
	for _, edge := range g.edges {
		if edge.source == nodename {
			startEdges = append(startEdges, edge)
		}
	}
	result := &Graph{
		nodes: map[string]Node{nodename: start},
		edges: startEdges,
	}
	// walk through the graph and add each node that we can reach from our start node
	for i := 0; i < len(result.edges); i++ {
		edge := result.edges[i]
		// check if target node is present
		if _, ok := result.nodes[edge.target]; ok {
			continue
		}
		// target has not been added yet, add it and its dependencies
		result.nodes[edge.target] = g.nodes[edge.target]
		// we modify the iterating slice here, but that is fine because it is only appending
		for _, e := range g.edges {
			if e.source == edge.target {
				result.edges = append(result.edges, e)
			}
		}
	}
	return result
}

// FromScanner reads data from the given scanner, building up the dependency tree.
func (g *Graph) FromScanner(scanner *bufio.Scanner, syntaxes ...*syntax.Syntax) (*Graph, error) {
	if len(syntaxes) == 0 {
		panic("FromScanner: At least one syntax required!")
	}
	scanner.Split(scanLineWithEscape)
	activeSyntaxes := make(map[*syntax.Syntax]struct{}, len(syntaxes))
	addEdge := func(s string, t string) { g.AddEdgeAndNodes(node(s), node(t)) }
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
				scanDependencies(line[prefIndex+len(syntax.EdgePrefix):suffixIndex], syntax, addEdge)
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

// scanDependencies adds the given dependency line with the given syntax as edges by calling the given addEdge function.
func scanDependencies(line string, syntax *syntax.Syntax, addEdge func(string, string)) {
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
					addEdge(source, target)
				}
			}
		}
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

// Synced embeds a graph.Graph and synchronizes read and write access with an embedded sync.RWMutex, making it concurrency-safe.
// It satisfies the graph.Interface and fmt.Stringer interfaces.
// The zero value is an uninitialized graph, use graph.NewSynced() to get an initialized graph.Synced.
//
// This implementation prefers fast(-ish) node lookups over edge removals because it keeps a map of all nodes ready.
type Synced struct {
	Graph
	sync.RWMutex
}

// NewSynced returns a freshly initialized, ready-to-use graph.Synced
// It takes an optional first parameter specifying the capacity of nodes that the Graph will contain, and an optional
// second uint specifying the number of edges it will hold.
func NewSynced(capacities ...uint) *Synced {
	return &Synced{Graph: *New(capacities...)}
}

// AddNode adds the node with edges to the nodes with the given target names to this graph.
func (g *Synced) AddNode(node Node, targetNames ...string) {
	g.Lock()
	defer g.Unlock()
	g.Graph.AddNode(node, targetNames...)
}

// AddNodes adds the given nodes to this graph.
func (g *Synced) AddNodes(nodes ...Node) {
	g.Lock()
	defer g.Unlock()
	g.Graph.AddNodes(nodes...)
}

// GetNode returns the Node with the specified name, or nil if the name couldn't be found in g.
func (g *Synced) GetNode(name string) Node {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetNode(name)
}

// GetNodes returns all Node objects saved in the Graph as a slice. If it is empty, an empty slice is returned.
func (g *Synced) GetNodes() []Node {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetNodes()
}

// RemoveNode removes the Node with the given name including its edges from the graph.
// Returns false if the graph didn't have a matching Node, true otherwise.
func (g *Synced) RemoveNode(name string) bool {
	g.Lock()
	defer g.Unlock()
	return g.Graph.RemoveNode(name)
}

// AddEdge adds an edge from the source Node to the target Node to the graph.
func (g *Synced) AddEdge(source, target string) {
	g.Lock()
	defer g.Unlock()
	g.Graph.AddEdge(source, target)
}

// AddEdgeAndNodes adds a new edge from source to target to the Graph. The nodes are added to the Graph if they were not present.
func (g *Synced) AddEdgeAndNodes(source, target Node) {
	g.Lock()
	defer g.Unlock()
	g.Graph.AddEdgeAndNodes(source, target)
}

// HasEdge returns true if g has an edge from the source Node to the target Node, false otherwise.
func (g *Synced) HasEdge(source, target string) bool {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.HasEdge(source, target)
}

// RemoveEdge removes the edge from the source Node to the target Node.
// Returns false if the graph didn't have an edge from source to target, true otherwise.
func (g *Synced) RemoveEdge(source, target string) bool {
	g.Lock()
	defer g.Unlock()
	return g.Graph.RemoveEdge(source, target)
}

// GetDependencies returns a slice containing all dependencies of the Node with the given string.
// The graph.Synced contains an edge from the Node to each item in the returned dependencies.
func (g *Synced) GetDependencies(node string) []Node {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetDependencies(node)
}

// GetDependants returns a slice containing all dependants of the Node with the given string.
// The graph.Synced contains an edge from each item in dependants to the Node.
func (g *Synced) GetDependants(node string) []Node {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetDependants(node)
}

// Copy returns a deep copy of the graph.Synced.
func (g *Synced) Copy() Interface {
	g.RLock()
	defer g.RUnlock()
	return &Synced{Graph: *g.Graph.Copy().(*Graph)}
}

// GetDependencyGraph builds the dependency graph for the node. Returns nil if no node with the given name was found in g.
// If read/write access to the dependency graph shall be thread-safe as well you need to embed it in a Synced!
func (g *Synced) GetDependencyGraph(nodename string) *Graph {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetDependencyGraph(nodename)
}

// FromScanner reads data from the given scanner, building up the dependency tree.
// This uses multiple workers to concurrently write the read edges.
func (g *Synced) FromScanner(scanner *bufio.Scanner, syntaxes ...*syntax.Syntax) (*Synced, error) {
	if len(syntaxes) == 0 {
		panic("FromScanner: At least one syntax required!")
	}
	scanner.Split(scanLineWithEscape)
	activeSyntaxes := make(map[*syntax.Syntax]struct{}, len(syntaxes))
	addEdge := func(s string, t string) { g.AddEdgeAndNodes(node(s), node(t)) }
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
				scanDependencies(task.line, task.syntax, addEdge)
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
func (g *Synced) String() string {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.String()
}
