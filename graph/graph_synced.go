/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package graph

import (
	"bufio"
	"github.com/SimplicityApks/depgrapher/syntax"
	"runtime"
	"strings"
	"sync"
)

// Synced embeds a graph.Graph and synchronizes read and write access with an embedded sync.RWMutex, making it concurrency-safe.
// It satisfies the graph.Interface and fmt.Stringer interfaces.
// The zero value is an uninitialized graph, use graph.NewSynced() to get an initialized graph.Synced.
//
// This implementation prefers fast node lookups and edge removals over dependency lookups.
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
//
// This operation takes constant time, O(1) (but proportional to the number of targetsNames).
func (g *Synced) AddNode(node Node, targetNames ...string) {
	g.Lock()
	defer g.Unlock()
	g.Graph.AddNode(node, targetNames...)
}

// AddNodes adds the given nodes to this graph.
//
// This operation takes constant time, O(1) (but proportional to the number of new nodes).
func (g *Synced) AddNodes(nodes ...Node) {
	g.Lock()
	defer g.Unlock()
	g.Graph.AddNodes(nodes...)
}

// GetNode returns the Node with the specified name, or nil if the name couldn't be found in g.
//
// This operation takes constant time, O(1).
func (g *Synced) GetNode(name string) Node {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetNode(name)
}

// GetNodes returns all Node objects saved in the Graph as a slice. If it is empty, an empty slice is returned.
//
// This operation takes time proportional to the number of nodes in g, O(n).
func (g *Synced) GetNodes() []Node {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetNodes()
}

// RemoveNode removes the Node with the given name including its edges from the graph.
// Returns false if the graph didn't have a matching Node, true otherwise.
//
// This operation takes time proportional to the number of nodes in g, O(n).
func (g *Synced) RemoveNode(name string) bool {
	g.Lock()
	defer g.Unlock()
	return g.Graph.RemoveNode(name)
}

// AddEdge adds an edge from the source Node to the target Node to the graph.
//
// This operation takes constant time, O(1).
func (g *Synced) AddEdge(source, target string) {
	g.Lock()
	defer g.Unlock()
	g.Graph.AddEdge(source, target)
}

// AddEdgeAndNodes adds a new edge from source to target to the Graph. The nodes are added to the Graph if they were not present.
//
// This operation takes constant time, O(1).
func (g *Synced) AddEdgeAndNodes(source, target Node) {
	g.Lock()
	defer g.Unlock()
	g.Graph.AddEdgeAndNodes(source, target)
}

// HasEdge returns true if g has an edge from the source Node to the target Node, false otherwise.
//
// This operation takes constant time, O(1).
func (g *Synced) HasEdge(source, target string) bool {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.HasEdge(source, target)
}

// RemoveEdge removes the edge from the source Node to the target Node.
// Returns false if the graph didn't have an edge from source to target, true otherwise.
//
// This operation takes constant time, O(1).
func (g *Synced) RemoveEdge(source, target string) bool {
	g.Lock()
	defer g.Unlock()
	return g.Graph.RemoveEdge(source, target)
}

// GetDependencies returns a slice containing all dependencies of the Node with the given string.
// The graph.Synced contains an edge from the Node to each item in the returned dependencies.
//
// This operation takes time proportional to the number of nodes in g, O(n).
func (g *Synced) GetDependencies(node string) []Node {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetDependencies(node)
}

// GetDependants returns a slice containing all dependants of the Node with the given string.
// The graph.Synced contains an edge from each item in dependants to the Node.
//
// This operation takes time proportional to the number of nodes in g, O(n).
func (g *Synced) GetDependants(node string) []Node {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.GetDependants(node)
}

// Copy returns a deep copy of the graph.Synced.
//
// This operation takes time proportional to the sum of the number of nodes and the number of edges in g, O(n+e).
func (g *Synced) Copy() Interface {
	g.RLock()
	defer g.RUnlock()
	return &Synced{Graph: *g.Graph.Copy().(*Graph)}
}

// GetDependencyGraph builds the dependency graph for the node. Returns nil if no node with the given name was found in g.
// If read/write access to the dependency graph shall be thread-safe as well you need to embed it in a Synced!
//
// This operation takes time proportional to the product of the number of nodes and the number of edges in g, O(n*e).
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
//
// This operation takes time proportional to the number of edges in g, O(e).
func (g *Synced) String() string {
	g.RLock()
	defer g.RUnlock()
	return g.Graph.String()
}
