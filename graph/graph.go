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
	"strings"
)

// Node represents a single data point stored in a Graph. Its String() method should return a unique string identifier.
// The String() method call should be fast (ideally inline), as it will be called often!
type Node fmt.Stringer

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
// This implementation prefers fast node lookups and edge removals over dependency lookups.
type Graph struct {
	nodes map[string]Node
	edges map[edge]struct{}
}

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

// New returns a freshly initialized, ready-to-use Graph.
// It takes an optional first parameter specifying the capacity of nodes that the Graph will contain, and an optional
// second uint specifying the number of edges it will hold.
func New(capacities ...uint) *Graph {
	switch len(capacities) {
	case 0:
		return &Graph{nodes: make(map[string]Node), edges: make(map[edge]struct{})}
	case 1:
		return &Graph{nodes: make(map[string]Node, capacities[0]), edges: make(map[edge]struct{})}
	case 2:
		return &Graph{nodes: make(map[string]Node, capacities[0]), edges: make(map[edge]struct{}, capacities[1])}
	default:
		panic("More than 2 capacity parameters for graph.New")
	}
}

// AddNode adds the node with edges to the nodes with the given target names to this graph.
//
// This operation takes constant time, O(1) (but proportional to the number of targetsNames).
func (g *Graph) AddNode(node Node, targetNames ...string) {
	nodeName := node.String()
	g.nodes[nodeName] = node
	for _, targetName := range targetNames {
		if _, ok := g.nodes[targetName]; !ok {
			panic(errors.New("AddNode: target node with name " + targetName + " not present in Graph"))
		}
		g.edges[edge{source: nodeName, target: targetName}] = struct{}{}
	}
}

// AddNodes adds the given nodes to this graph.
//
// This operation takes constant time, O(1) (but proportional to the number of new nodes).
func (g *Graph) AddNodes(nodes ...Node) {
	for _, node := range nodes {
		g.nodes[node.String()] = node
	}
}

// GetNode returns the Node with the specified name, or nil if the name couldn't be found in g.
//
// This operation takes constant time, O(1).
func (g *Graph) GetNode(name string) Node {
	return g.nodes[name]
}

// GetNodes returns all Node objects saved in the Graph as a slice. If it is empty, an empty slice is returned.
//
// This operation takes time proportional to the number of nodes in g, O(n).
func (g *Graph) GetNodes() []Node {
	nodes := make([]Node, 0, len(g.nodes))
	for _, node := range g.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// RemoveNode removes the Node with the given name including its edges from the graph.
// Returns false if the graph didn't have a matching Node, true otherwise.
//
// This operation takes time proportional to the number of nodes in g, O(n).
func (g *Graph) RemoveNode(name string) bool {
	if _, ok := g.nodes[name]; !ok {
		return false
	}
	delete(g.nodes, name)
	for node := range g.nodes {
		delete(g.edges, edge{source: name, target: node})
		delete(g.edges, edge{source: node, target: name})
	}
	return true
}

// AddEdge adds an edge from the source Node to the target Node to the graph.
//
// This operation takes constant time, O(1).
func (g *Graph) AddEdge(source, target string) {
	if _, ok := g.nodes[source]; !ok {
		panic("AddEdge: source Node not present in Graph!")
	}
	if _, ok := g.nodes[target]; !ok {
		panic("AddEdge: source Node not present in Graph!")
	}
	g.edges[edge{source: source, target: target}] = struct{}{}
}

// AddEdgeAndNodes adds a new edge from source to target to the Graph. The nodes are added to the Graph if they were not present.
//
// This operation takes constant time, O(1).
func (g *Graph) AddEdgeAndNodes(source, target Node) {
	sourceName, targetName := source.String(), target.String()
	// add the nodes if they aren't already in place
	if _, ok := g.nodes[sourceName]; !ok {
		g.nodes[sourceName] = source
	}
	if _, ok := g.nodes[targetName]; !ok {
		g.nodes[targetName] = target
	}
	g.edges[edge{source: sourceName, target: targetName}] = struct{}{}
}

// HasEdge returns true if g has an edge from the source Node to the target Node, false otherwise.
//
// This operation takes constant time, O(1).
func (g *Graph) HasEdge(source, target string) bool {
	_, ok := g.edges[edge{source: source, target: target}]
	return ok
}

// RemoveEdge removes the edge from the source Node to the target Node.
// Returns false if the graph didn't have an edge from source to target, true otherwise.
//
// This operation takes constant time, O(1).
func (g *Graph) RemoveEdge(source, target string) bool {
	e := edge{source: source, target: target}
	_, ok := g.edges[e]
	delete(g.edges, e)
	return ok
}

// GetDependencies returns a slice containing all dependencies of the Node with the given string.
// The Graph contains an edge from the Node to each item in the returned dependencies.
//
// This operation takes time proportional to the number of nodes in g, O(n).
func (g *Graph) GetDependencies(node string) []Node {
	var deps []Node
	for currentName, current := range g.nodes {
		if _, ok := g.edges[edge{source: node, target: currentName}]; ok {
			deps = append(deps, current)
		}
	}
	return deps
}

// GetDependants returns a slice containing all dependants of the Node with the given string.
// The Graph contains an edge from each item in dependants to the Node.
//
// This operation takes time proportional to the number of nodes in g, O(n).
func (g *Graph) GetDependants(node string) []Node {
	var deps []Node
	for currentName, current := range g.nodes {
		if _, ok := g.edges[edge{source: currentName, target: node}]; ok {
			deps = append(deps, current)
		}
	}
	return deps
}

// Copy returns a deep copy of the Graph.
//
// This operation takes time proportional to the sum of the number of nodes and the number of edges in g, O(n+e).
func (g *Graph) Copy() Interface {
	result := &Graph{
		nodes: make(map[string]Node, len(g.nodes)),
		edges: make(map[edge]struct{}, len(g.edges)),
	}
	for k, v := range g.nodes {
		result.nodes[k] = v
	}
	for e := range g.edges {
		result.edges[e] = struct{}{}
	}
	return result
}

// GetDependencyGraph builds the dependency graph for the node. Returns nil if no node with the given name was found in g.
//
// This operation takes time proportional to product of the number of nodes and the number of edges in g, O(n*e).
func (g *Graph) GetDependencyGraph(nodename string) *Graph {
	start, ok := g.nodes[nodename]
	if !ok {
		return nil
	}
	var edges []edge
	for n := range g.nodes {
		e := edge{source: nodename, target: n}
		if _, ok := g.edges[e]; ok {
			edges = append(edges, e)
		}
	}
	result := &Graph{
		nodes: map[string]Node{nodename: start},
		edges: map[edge]struct{}{},
	}
	// walk through the graph and add each node that we can reach from our start node
	for i := 0; i < len(edges); i++ {
		current := edges[i]
		result.edges[current] = struct{}{}
		// check if target node is present
		if _, ok := result.nodes[current.target]; ok {
			continue
		}
		// target has not been added yet, add it and its dependencies
		result.nodes[current.target] = g.nodes[current.target]
		// we modify the iterating slice here, but that is fine because it is only appending
		for targetNode := range g.nodes {
			e := edge{source: current.target, target: targetNode}
			if _, ok := g.edges[e]; ok {
				edges = append(edges, e)
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
//
// This operation takes time proportional to the number of edges in g, O(e).
func (g *Graph) String() string {
	if len(g.nodes) == 0 {
		return "{empty graph}"
	}
	var buffer bytes.Buffer
	for edge := range g.edges {
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
