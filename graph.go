package main

import (
	"bufio"
	"bytes"
	"strings"
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
}

func (g *Graph) AddEdge(sourceName string, targetName string) {
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

func (g *Graph) GetNode(name string) *Node {
	for _, node := range g.nodes {
		if node.name == name {
			return node
		}
	}
	return nil
}

func (g *Graph) GetDependencies(n *Node) []*Node {
	deps := make([]*Node, 0)
	for _, edge := range g.edges {
		if edge.source == n {
			deps = append(deps, edge.target)
		}
	}
	return deps
}

func (g *Graph) GetDependants(n *Node) []*Node {
	deps := make([]*Node, 0)
	for _, edge := range g.edges {
		if edge.target == n {
			deps = append(deps, edge.source)
		}
	}
	return deps
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
					g.AddEdge(source, target)
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
			g.scanDependencies(line[prefIndex+len(syntax.prefix):suffixIndex], syntax)
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
	for err == nil && len(token) > 0 && token[len(token)-1] == '\\' {
		secondAdvance, secondToken, secondErr := bufio.ScanLines(data, atEOF)
		advance += secondAdvance
		token = append(token, secondToken...)
		err = secondErr
	}
	return
}

func insertIntoString(s *string, offset int, insertion string) *string {
	var str string
	if s == nil {
		str = strings.Repeat(" ", offset) + insertion
	} else if offset > len(*s) {
		str = *s + strings.Repeat(" ", offset-len(*s)) + insertion
	} else {
		str = (*s)[:offset] + insertion + (*s)[offset:]
	}
	return &str
}

func printDepTreeLevel(g *Graph, n *Node, out []*string, rightOffset int) (modout []*string, height int, width int) {
	dependencies := g.GetDependencies(n)
	namelen := len(n.name) + 2
	if len(dependencies) == 0 {
		name := " " + n.name + " "
		if len(out) == 0 {
			name = strings.Repeat(" ", rightOffset) + name
			out = append(out, &name)
		} else {
			out[0] = insertIntoString(out[0], rightOffset, name)
		}
		// we are at the bottom of the graph
		return out, 1, namelen
	} else {
		if len(out) == 0 {
			out = append(out, nil)
		}
		modout = out
		out = out[1:]
		// we have some dependencies, let us print them
		var depWidth, locHeight, locWidth int
		for _, dep := range dependencies {
			out, locHeight, locWidth = printDepTreeLevel(g, dep, out, rightOffset+depWidth)
			depWidth += locWidth
			if height < locHeight {
				height = locHeight
			}
		}
		modout = append(modout[:1], out...)
		// print our name in the middle
		if namelen > depWidth {
			// shift everything below to the right
			shiftOffsetLeft := strings.Repeat(" ", (namelen-depWidth)/2)
			shiftOffsetRight := strings.Repeat(" ", (namelen-depWidth+1)/2)
			for _, strptr := range out {
				strptr = insertIntoString(insertIntoString(strptr, rightOffset+depWidth, shiftOffsetRight), rightOffset, shiftOffsetLeft)
			}
			modout[0] = insertIntoString(modout[0], rightOffset, " "+n.name+" ")
			return modout, height, len(n.name) + 2
		} else {
			modout[0] = insertIntoString(modout[0], rightOffset, strings.Repeat(" ", (depWidth-namelen)/2)+" "+n.name+" "+strings.Repeat(" ", (depWidth-namelen+1)/2))
			return modout, height, depWidth
		}
	}
}

// printDepTree pretty prints the dependency tree of the specified startNode to stdout
func printDepTree(g *Graph, startNode *Node) {
	out, _, _ := printDepTreeLevel(g, startNode, make([]*string, 1), 0)
	for _, lineptr := range out {
		println(*lineptr)
	}
}

// printFullDepTree prints the dependency tree of the whole graph to stdout
func printFullDepTree(g *Graph) {
	// make a copy of our graph
	fullGraph := *g
	for _, node := range g.nodes {
		// TODO add only edges where the graph is separated
		fullGraph.AddEdge("_all", node.name)
	}
	printDepTree(&fullGraph, fullGraph.GetNode("_all"))
}
