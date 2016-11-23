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

func (g *Graph) scanDependencies(line string, syntax Syntax) {
	infixIndex := strings.Index(line, syntax.EdgeInfix)
	sources := strings.Split(line[:infixIndex], syntax.SourceDelimiter)
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
func (g *Graph) FromScanner(scanner *bufio.Scanner, syntax Syntax) (*Graph, error) {
	scanner.Split(scanLineWithEscape)
	if syntax.GraphPrefix != "" {
		for scanner.Scan() {
			if scanner.Err() != nil {
				return g, scanner.Err()
			}
			if strings.Contains(scanner.Text(), syntax.GraphPrefix) {
				break
			}
		}
	}
	// for running concurrently, we'll add a pool of worker goroutines
	numWorkers := runtime.GOMAXPROCS(0)
	// we need to wait for our goroutines to finish
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(numWorkers)
	defer waitGroup.Wait()
	tasks := make(chan string, numWorkers)
	defer close(tasks)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer waitGroup.Done()
			for task := range tasks {
				g.scanDependencies(task, syntax)
			}
		}()
	}
	for scanner.Scan() {
		if scanner.Err() != nil {
			return g, scanner.Err()
		}
		line := scanner.Text()
		if syntax.GraphSuffix != "" && strings.Contains(line, syntax.GraphSuffix) {
			break
		}
		prefIndex := strings.Index(line, syntax.EdgePrefix)
		infixIndex := strings.Index(line, syntax.EdgeInfix)
		suffixIndex := strings.LastIndex(line, syntax.EdgeSuffix)
		if prefIndex >= 0 && infixIndex >= 0 && suffixIndex >= 0 {
			tasks <- line[prefIndex+len(syntax.EdgePrefix) : suffixIndex]
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
		nodes: make([]*Node, 0),
		edges: make([]*edge, 0),
	}
}

func scanLineWithEscape(data []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanLines(data, atEOF)
	for err == nil && len(token) > 0 && token[len(token)-1] == '\\' {
		secondAdvance, secondToken, secondErr := bufio.ScanLines(data, atEOF)
		advance += secondAdvance
		token = append(token, secondToken...)
		err = secondErr
	}
	return
}
