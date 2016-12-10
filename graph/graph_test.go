/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package graph

import (
	"bufio"
	"github.com/SimplicityApks/depgrapher/syntax"
	"strconv"
	"strings"
	"testing"
)

// Most Tests and benchmarks use a level graph returned by setupLevelGraph with the following number of levels.
// The number of levels should be at least 3, increasing it will slow down tests exponentially!
const (
	BENCH_GRAPH_LEVELS = 10
	TEST_GRAPH_LEVELS  = 3
)

// Test setup utilities and helper methods

// The tests use int nodes most of the time for simplicity
type intnode uint

func (i intnode) String() string {
	return strconv.Itoa(int(i))
}

// setupLevelGraph returns a Graph with the following structure:
//              1                // level 0
//         /         \
//        V           V
//      2             3 		 // level 1
//    /|\  \       / | \   \
//   V V V  V     V  V  V   V
//  4  5  6  7  &4  &5  &6  &7   // level 2
//  ....                         // level levels-1
// so that each node has edges to all nodes in the next level.
func setupLevelGraph(levels uint) *Graph {
	var nodeCount uint = (1 << levels) - 1
	// every node has edges to all nodes in the next level
	// edgeCount = sum from n:=0 to levels-1 of (2^n*(2^(n+1))) - 2^levels
	//  		 = sum from n:=0 to levels-2 of 2^(2n) * 2
	//           = 2* (1-4^(levels-1)) / (1-4)
	var edgeCount uint = ((1 << (2 * (levels - 1))) - 1) * 2 / 3
	g := New(nodeCount, edgeCount)

	if levels == 0 {
		return g
	} else if levels == 1 {
		g.AddNode(intnode(1))
		return g
	}
	// build it from the bottom up so we can use AddNode with adding edges directly
	prevLevelNodes := []string{}
	for level := levels - 1; level > 0; level-- {
		levelNodes := make([]string, 0, 1<<level)
		for nodeNumber := 1 << level; nodeNumber < (1 << (level + 1)); nodeNumber++ {
			g.AddNode(intnode(nodeNumber), prevLevelNodes...)
			levelNodes = append(levelNodes, strconv.Itoa(nodeNumber))
		}
		prevLevelNodes = levelNodes
	}
	// add the last node manually so we can use level>0 above
	g.AddNode(intnode(1), "2", "3")
	return g
}

func TestGraph_AddNode(t *testing.T) {
	g := New()
	n1 := intnode(1)
	g.AddNode(n1)
	if len(g.GetNodes()) != 1 || g.GetNodes()[0] != n1 {
		t.FailNow()
	}
	n2 := intnode(2)
	g.AddNode(n2, "1")
	if len(g.GetNodes()) != 2 {
		t.FailNow()
	}
	if !g.HasEdge("2", "1") || g.HasEdge("1", "2") {
		t.FailNow()
	}
	g.AddNode(intnode(3), "3")
	if !g.HasEdge("3", "3") {
		t.Error("AddNode didn't add node with edge to itself")
	}
}

func TestGraph_AddNodes(t *testing.T) {
	g := New()
	g.AddNodes()
	if len(g.GetNodes()) != 0 {
		t.FailNow()
	}
	n1 := intnode(1)
	g.AddNodes(n1)
	if len(g.GetNodes()) != 1 || g.GetNodes()[0] != n1 {
		t.FailNow()
	}
	g.AddNodes(intnode(2), intnode(3))
	if len(g.GetNodes()) != 3 {
		t.FailNow()
	}
	if g.HasEdge("1", "2") || g.HasEdge("2", "1") || g.HasEdge("1", "3") || g.HasEdge("3", "1") {
		t.Error("AddNodes added unexpected edges")
	}
}

func TestGraph_GetNode(t *testing.T) {
	var levels uint = TEST_GRAPH_LEVELS
	g := setupLevelGraph(levels)
	if g.GetNode("nonexistent") != nil {
		t.Error("GetNode returned a value for a not existing node")
	}
	for i := 1; i < (1 << levels); i++ {
		n1 := g.GetNode(strconv.Itoa(i))
		if n1 == nil {
			t.Errorf("GetNode returned nil for node %d", i)
			continue
		}
		if int(n1.(intnode)) != i {
			t.Errorf("GetNode returned wrong node number %d: %d", i, int(n1.(intnode)))
		}
	}
}

func TestGraph_GetNodes(t *testing.T) {
	var levels uint = TEST_GRAPH_LEVELS
	g := setupLevelGraph(levels)
	nodes := g.GetNodes()
	if len(nodes) != (1<<levels)-1 {
		t.FailNow()
	}
NODELOOP:
	for i := 1; i < (1 << levels); i++ {
		for _, node := range nodes {
			if int(node.(intnode)) == i {
				continue NODELOOP
			}
		}
		t.Errorf("GetNodes didn't contain node %d", i)
	}
}

func TestGraph_RemoveNode(t *testing.T) {
	var levels uint = TEST_GRAPH_LEVELS
	g := setupLevelGraph(levels)
	if g.RemoveNode("nonexistent") {
		t.Error("RemoveNode returned true for not existing node")
	}
	for i := 1; i < (1 << levels); i++ {
		if !g.RemoveNode(strconv.Itoa(i)) {
			t.Errorf("RemoveNode returned false for existing node %d", i)
		}
		if g.GetNode(strconv.Itoa(i)) != nil {
			t.Errorf("RemoveNode didn't remove node %d", i)
		}
	}
	if len(g.GetNodes()) != 0 {
		t.Error("RemoveNode didn't remove all nodes")
	}
}

func TestGraph_AddEdge(t *testing.T) {
	g := New(3, 3)
	g.AddNodes(intnode(1), intnode(2), intnode(3))
	g.AddEdge("1", "2")
	if !g.HasEdge("1", "2") {
		t.Fail()
	}
	g.AddEdge("1", "3")
	if !g.HasEdge("1", "3") {
		t.Fail()
	}
	if g.HasEdge("2", "3") || g.HasEdge("3", "2") || g.HasEdge("3", "1") {
		t.Error("AddEdge added wrong edges")
	}
	g.AddEdge("1", "1")
	if !g.HasEdge("1", "1") {
		t.Error("AddEdge didn't add edge to same node")
	}
}

func TestGraph_AddEdgeAndNodes(t *testing.T) {
	g := New()
	n := intnode(1)
	g.AddEdgeAndNodes(n, intnode(2))
	if !g.HasEdge("1", "2") {
		t.Fail()
	}
	g.AddEdgeAndNodes(n, intnode(3))
	if !g.HasEdge("1", "3") {
		t.Fail()
	}
	if g.HasEdge("2", "3") || g.HasEdge("3", "2") || g.HasEdge("3", "1") {
		t.Error("AddEdge added wrong edges")
	}
	g.AddEdgeAndNodes(n, n)
	if !g.HasEdge("1", "1") {
		t.Error("Couldn't add edge to same node")
	}

	g2 := setupLevelGraph(3)
	n1, n7 := g2.GetNode("1"), g2.GetNode("7")
	if n1 == nil || n7 == nil {
		t.Fatal("Couldn't find nodes in levelgraph")
	}
	g2.AddEdgeAndNodes(n1.(intnode), n7.(intnode))
	if len(g2.GetNodes()) != 7 {
		t.Error("AddEdge added nodes unexpectedly")
	}
	if !g2.HasEdge("1", "7") {
		t.Fail()
	}
}

func TestGraph_HasEdge(t *testing.T) {
	var levels, level uint = TEST_GRAPH_LEVELS, 0
	g := setupLevelGraph(levels)
	// check not existing nodes
	if g.HasEdge("1", "nonexistent") || g.HasEdge("nonexistent", "1") || g.HasEdge("nonexistent", "nonexistent2") {
		t.Error("HasEdge returned true for a not existing node")
	}
	if g.HasEdge("1", "1") {
		t.Error("HasEdge returned true for an not added edge from a node to itself")
	}
	// loop through all edges
	for ; level < levels-1; level++ {
		for nodeNumber := 1 << level; nodeNumber < (1 << (level + 1)); nodeNumber++ {
			for nodeBelow := 1 << (level + 1); nodeBelow < (1 << (level + 2)); nodeBelow++ {
				if !g.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeBelow)) {
					t.Errorf("HasEdge returned false for %d=>%d", nodeNumber, nodeBelow)
				}
			}
			for nodeAbove := 1; nodeAbove < nodeNumber; nodeAbove++ {
				if g.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeAbove)) {
					t.Errorf("HasEdge returned true for %d=>%d", nodeNumber, nodeAbove)
				}
			}
		}
	}
}

func TestGraph_RemoveEdge(t *testing.T) {
	var levels, level uint = TEST_GRAPH_LEVELS, 0
	g := setupLevelGraph(levels)
	// check not existing nodes
	if g.RemoveEdge("1", "nonexistent") || g.RemoveEdge("nonexistent", "1") || g.RemoveEdge("nonexistent", "nonexistent2") {
		t.Error("RemoveEdge returned true for a not existing node")
	}
	// loop through all edges
	for ; level < levels-1; level++ {
		for nodeNumber := 1 << level; nodeNumber < (1 << (level + 1)); nodeNumber++ {
			for nodeBelow := 1 << (level + 1); nodeBelow < (1 << (level + 2)); nodeBelow++ {
				if !g.RemoveEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeBelow)) {
					t.Errorf("RemoveEdge returned false for %d=>%d", nodeNumber, nodeBelow)
				}
				if g.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeBelow)) {
					t.Errorf("RemoveEdge didn't remove edge %d=>%d", nodeNumber, nodeBelow)
				}
			}
			for nodeAbove := 1; nodeAbove < nodeNumber; nodeAbove++ {
				if g.RemoveEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeAbove)) {
					t.Errorf("RemoveEdge returned true for %d=>%d", nodeNumber, nodeAbove)
				}
			}
		}
	}
}

func TestGraph_GetDependencies(t *testing.T) {
	var levels, level uint = TEST_GRAPH_LEVELS, 0
	g := setupLevelGraph(levels)
	// check nonexisting node
	if len(g.GetDependencies("nonexistent")) != 0 {
		t.Error("GetDependencies returned dependencies for a non-existing node")
	}
	// loop through all edges except the last level
	for ; level < levels-1; level++ {
		for nodeNumber := 1 << level; nodeNumber < (1 << (level + 1)); nodeNumber++ {
			deps := g.GetDependencies(strconv.Itoa(nodeNumber))
			if len(deps) != 1<<(level+1) {
				t.Errorf("GetDependencies didn't return the correct number of dependencies for node %d", nodeNumber)
			}
			// check each node in deps
		NODELOOP:
			for nodeBelow := 1 << (level + 1); nodeBelow < (1 << (level + 2)); nodeBelow++ {
				for _, node := range deps {
					if int(node.(intnode)) == nodeBelow {
						continue NODELOOP
					}
				}
				t.Errorf("GetDependencies didn't contain node %d", nodeBelow)
			}
		}
	}
	// check the last level with no dependencies
	for nodeNumber := 1<<levels - 1; nodeNumber < (1 << levels); nodeNumber++ {
		if len(g.GetDependencies(strconv.Itoa(nodeNumber))) != 0 {
			t.Errorf("GetDependencies didn't return an empty slice for node %d", nodeNumber)
		}
	}
}

func TestGraph_GetDependants(t *testing.T) {
	var levels, level uint = TEST_GRAPH_LEVELS, 0
	g := setupLevelGraph(levels)
	// check nonexisting node
	if len(g.GetDependants("nonexistent")) != 0 {
		t.Error("GetDependants returned dependants for a non-existing node")
	}
	// loop through all edges except the first level
	for level = 1; level < levels; level++ {
		for nodeNumber := 1 << level; nodeNumber < (1 << (level + 1)); nodeNumber++ {
			deps := g.GetDependants(strconv.Itoa(nodeNumber))
			if len(deps) != 1<<(level-1) {
				t.Errorf("GetDependants didn't return the correct number of dependants for node %d", nodeNumber)
			}
			// check each node in deps
		NODELOOP:
			for nodeAbove := 1 << (level - 1); nodeAbove < (1 << (level)); nodeAbove++ {
				for _, node := range deps {
					if int(node.(intnode)) == nodeAbove {
						continue NODELOOP
					}
				}
				t.Errorf("GetDependants didn't contain node %d", nodeAbove)
			}
		}
	}
	// check the first level with no dependencies
	if len(g.GetDependants(strconv.Itoa(1))) != 0 {
		t.Errorf("GetDependants didn't return an empty slice for node 1")
	}
}

func TestGraph_Copy(t *testing.T) {
	var levels, level uint = TEST_GRAPH_LEVELS, 0
	g := setupLevelGraph(levels)
	gCopy := g.Copy()
	// remove a node so we are sure it is a copy
	g.RemoveNode("1")
	if len(gCopy.GetNodes()) != (1<<levels)-1 {
		t.Error("Copy didn't contain the expected number of nodes")
	}
	for i := 1; i < (1 << levels); i++ {
		n1 := gCopy.GetNode(strconv.Itoa(i))
		if n1 == nil || int(n1.(intnode)) != i {
			t.Errorf("Copy didn't contain node %d", i)
			continue
		}
	}
	for ; level < levels-1; level++ {
		for nodeNumber := 1 << level; nodeNumber < (1 << (level + 1)); nodeNumber++ {
			for nodeBelow := 1 << (level + 1); nodeBelow < (1 << (level + 2)); nodeBelow++ {
				if !gCopy.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeBelow)) {
					t.Errorf("Copy didn't contain edge %d=>%d", nodeNumber, nodeBelow)
				}
			}
			for nodeAbove := 1; nodeAbove < nodeNumber; nodeAbove++ {
				if gCopy.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeAbove)) {
					t.Errorf("Copy contained unexpected edge %d=>%d", nodeNumber, nodeAbove)
				}
			}
		}
	}
}

func TestGraph_GetDependencyGraph(t *testing.T) {
	var levels, level uint = TEST_GRAPH_LEVELS, 0
	g := setupLevelGraph(levels)
	// test nonexistent node
	if g.GetDependencyGraph("nonexistent") != nil {
		t.Error("GetDependencyGraph didn't return nil for a node not in the graph")
	}
	// add one more node
	g.AddNode(intnode(0), "1")
	// add an edge to the same loop (possible infinite recursion!)
	g.AddEdge("1", "1")
	// get the subgraph, which should equal the starting level graph
	depgraph := g.GetDependencyGraph("1")
	if len(depgraph.GetNodes()) != (1<<levels)-1 {
		t.Error("GetDependencyGraph didn't contain the expected number of nodes")
	}
	if !depgraph.HasEdge("1", "1") {
		t.Error("GetDependencyGraph didn't contain edge from node to itself")
	}
	for i := 1; i < (1 << levels); i++ {
		n1 := depgraph.GetNode(strconv.Itoa(i))
		if n1 == nil || int(n1.(intnode)) != i {
			t.Errorf("GetDependencyGraph didn't contain node %d", i)
			continue
		}
	}
	for ; level < levels-1; level++ {
		for nodeNumber := 1 << level; nodeNumber < (1 << (level + 1)); nodeNumber++ {
			for nodeBelow := 1 << (level + 1); nodeBelow < (1 << (level + 2)); nodeBelow++ {
				if !depgraph.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeBelow)) {
					t.Errorf("GetDependencyGraph didn't contain edge %d=>%d", nodeNumber, nodeBelow)
				}
			}
			for nodeAbove := 1; nodeAbove < nodeNumber; nodeAbove++ {
				if depgraph.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeAbove)) {
					t.Errorf("GetDependencyGraph contained unexpected edge %d=>%d", nodeNumber, nodeAbove)
				}
			}
		}
	}
}

func TestGraph_FromScanner(t *testing.T) {
	var levels, level uint = 3, 0
	// test empty string
	gEmpty, err := New().FromScanner(bufio.NewScanner(strings.NewReader("")), syntax.Makefile)
	if err != nil || len(gEmpty.GetNodes()) != 0 {
		t.Error("FromScanner returned a non-empty graph for an empty scanner")
	}
	// build a levelGraph with 3 levels
	graphString := "1:2 3 \n 2 3: 4 5 6 7\n"
	g, err := New().FromScanner(bufio.NewScanner(strings.NewReader(graphString)), syntax.Makefile)
	if err != nil {
		t.Error("FromScanner returned an error for a levelGraph")
	}
	if len(g.GetNodes()) != 7 {
		t.Errorf("FromScanner returned %d nodes instead of 7 for a levelGraph", len(gEmpty.GetNodes()))
	}
	for i := 1; i < (1 << levels); i++ {
		n1 := g.GetNode(strconv.Itoa(i))
		if n1 == nil || n1.String() != strconv.Itoa(i) {
			t.Errorf("GetDependencyGraph didn't contain node %d", i)
			continue
		}
	}
	for ; level < levels-1; level++ {
		for nodeNumber := 1 << level; nodeNumber < (1 << (level + 1)); nodeNumber++ {
			for nodeBelow := 1 << (level + 1); nodeBelow < (1 << (level + 2)); nodeBelow++ {
				if !g.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeBelow)) {
					t.Errorf("GetDependencyGraph didn't contain edge %d=>%d", nodeNumber, nodeBelow)
				}
			}
			for nodeAbove := 1; nodeAbove < nodeNumber; nodeAbove++ {
				if g.HasEdge(strconv.Itoa(nodeNumber), strconv.Itoa(nodeAbove)) {
					t.Errorf("GetDependencyGraph contained unexpected edge %d=>%d", nodeNumber, nodeAbove)
				}
			}
		}
	}
}

// BENCHMARKS

func BenchmarkGraph_AddNode(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	const nodeOffset = 1 << BENCH_GRAPH_LEVELS
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.AddNode(intnode(nodeOffset + i))
	}
}

func BenchmarkGraph_AddNode_dependencies(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	const nodeOffset = 1 << BENCH_GRAPH_LEVELS
	nodes := g.GetNodes()
	deps := make([]string, len(nodes))
	for i, node := range nodes {
		deps[i] = node.String()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.AddNode(intnode(nodeOffset+i), deps...)
	}
}

func BenchmarkGraph_AddNodes(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	const nodeOffset = 1 << BENCH_GRAPH_LEVELS
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.AddNodes(intnode(nodeOffset+2*i), intnode(nodeOffset+2*i+1))
	}
}

func BenchmarkGraph_GetNode(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// run GetNode on the node on the bottom right of the graph (worst case in the levelGraph)
	nodename := strconv.Itoa(1<<BENCH_GRAPH_LEVELS - 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetNode(nodename)
	}
}

func BenchmarkGraph_GetNodes(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetNodes()
	}
}

func BenchmarkGraph_RemoveNode(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// run RemoveNode on the nodes in the bottom level of the graph (worst case in the levelGraph, but doesn't remove dependencies)
	const nodecount = 1 << (BENCH_GRAPH_LEVELS - 1)
	nodenames := make([]string, 0, nodecount)
	for n := nodecount; n < 2*nodecount; n++ {
		nodenames = append(nodenames, strconv.Itoa(n))
	}
	node := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if node == nodecount {
			// we removed all nodes in the level, restore it
			b.StopTimer()
			g = setupLevelGraph(BENCH_GRAPH_LEVELS)
			node = 0
			b.StartTimer()
		}
		g.RemoveNode(nodenames[node])
		node++
	}
}

// remove a node with a ton of dependencies
func BenchmarkGraph_RemoveNode_dependencies(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// run RemoveNode on the nodes just above the bottom level of the graph (worst case in the levelGraph)
	const nodecount = 1 << (BENCH_GRAPH_LEVELS - 2)
	nodenames := make([]string, 0, nodecount)
	for n := nodecount; n < 2*nodecount; n++ {
		nodenames = append(nodenames, strconv.Itoa(n))
	}
	node := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if node == nodecount {
			// we removed all nodes in the level, restore it
			b.StopTimer()
			g = setupLevelGraph(BENCH_GRAPH_LEVELS)
			node = 0
			b.StartTimer()
		}
		g.RemoveNode(nodenames[node])
		node++
	}
}

func BenchmarkGraph_AddEdge(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// run AddEdge on the node on the bottom level of the graph (worst case in the levelGraph)
	const nodecount = 1 << (BENCH_GRAPH_LEVELS - 1)
	nodenames := make([]string, 0, nodecount)
	for n := nodecount; n < 2*nodecount; n++ {
		nodenames = append(nodenames, strconv.Itoa(n))
	}
	node := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if node == nodecount {
			// we added edges for all nodes in the level, restore it
			b.StopTimer()
			g = setupLevelGraph(BENCH_GRAPH_LEVELS)
			node = 0
			b.StartTimer()
		}
		g.AddEdge(nodenames[node], "1")
		node++
	}
}

func BenchmarkGraph_AddEdgeAndNodes(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// run AddEdge on the node on the bottom right of the graph (worst case in the levelGraph)
	var node uint = 1 << (BENCH_GRAPH_LEVELS - 1)
	startnode := intnode(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if node == (1 << BENCH_GRAPH_LEVELS) {
			// we added edges for all nodes in the level, restore it
			b.StopTimer()
			g = setupLevelGraph(BENCH_GRAPH_LEVELS)
			node = 1 << (BENCH_GRAPH_LEVELS - 1)
			b.StartTimer()
		}
		g.AddEdgeAndNodes(intnode(node), startnode)
		node++
	}
}

func BenchmarkGraph_HasEdge(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// run HasEdge on the edge on the bottom right of the graph (worst case in the levelGraph)
	sourcename := strconv.Itoa(1<<(BENCH_GRAPH_LEVELS-1) - 1)
	targetname := strconv.Itoa(1<<BENCH_GRAPH_LEVELS - 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.HasEdge(sourcename, targetname)
	}
}

func BenchmarkGraph_RemoveEdge(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// run RemoveEdge on the edges from the node on the bottom right to the bottom level of the graph (worst case in the levelGraph)
	const nodecount = 1 << (BENCH_GRAPH_LEVELS - 1)
	source := strconv.Itoa(nodecount - 1)
	nodenames := make([]string, 0, nodecount)
	for n := nodecount; n < 2*nodecount; n++ {
		nodenames = append(nodenames, strconv.Itoa(n))
	}
	target := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if target == nodecount {
			// we removed all edges on the bottom, restore the graph
			b.StopTimer()
			g = setupLevelGraph(BENCH_GRAPH_LEVELS)
			target = 0
			b.StartTimer()
		}
		g.RemoveEdge(source, nodenames[target])
		target++
	}
}

func BenchmarkGraph_GetDependencies(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// call GetDependants on the node just above the bottom left (worst case)
	nodename := strconv.Itoa((1 << BENCH_GRAPH_LEVELS) - 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetDependencies(nodename)
	}
}

func BenchmarkGraph_GetDependants(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	// call GetDependants on the node on the bottom right (worst case)
	nodename := strconv.Itoa((1 << BENCH_GRAPH_LEVELS) - 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetDependants(nodename)
	}
}

func BenchmarkGraph_GetDependencyGraph(b *testing.B) {
	g := setupLevelGraph(BENCH_GRAPH_LEVELS)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetDependencyGraph("1")
	}
}

func BenchmarkGraph_FromScanner(b *testing.B) {
	// build a levelGraph with 5 levels
	const graphString = "1 : 2 3 \n 2 3 : 4 5 6 7\n 4 5 6 7 : 8 9 10 11 12 13 14 15 \n 8 9 10 11 12 13 14 15 : 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31"
	scanner := bufio.NewScanner(strings.NewReader(graphString))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New().FromScanner(scanner, syntax.Makefile)
		scanner = bufio.NewScanner(strings.NewReader(graphString))
	}
}
