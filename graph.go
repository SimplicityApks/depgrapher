package main

import (
	"bufio"
	"bytes"
	"log"
	"strings"
	"sync"
	"time"
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

// executes g.scanDependencies for each task received from tasks
func scanDependenciesGo(tasks chan string, g *Graph, syntax Syntax) {
	for task := range tasks {
		g.scanDependencies(task, syntax)
	}
	waitGroup.Done()
}

var waitGroup sync.WaitGroup

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

// FromScanner reads data from the given scanner, building up the dependency tree.
func (g *Graph) FromScanner(scanner *bufio.Scanner, syntax Syntax) (*Graph, error) {
	defer timeTrack(time.Now(), "FromScanner")
	scanner.Split(scanLineWithEscape)
	// for running concurrently, we'll add a pool of worker goroutines
	numWorkers := 4
	tasks := make(chan string, numWorkers)
	// we need to wait for our goroutines to finish
	waitGroup = sync.WaitGroup{}
	waitGroup.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go scanDependenciesGo(tasks, g, syntax)
	}
	for scanner.Scan() {
		if scanner.Err() != nil {
			return g, scanner.Err()
		}
		line := string(scanner.Text())
		prefIndex := strings.Index(line, syntax.prefix)
		infixIndex := strings.Index(line, syntax.infix)
		suffixIndex := strings.LastIndex(line, syntax.suffix)
		if prefIndex >= 0 && infixIndex >= 0 && suffixIndex >= 0 {
			tasks <- line[prefIndex+len(syntax.prefix) : suffixIndex]
		}
	}
	close(tasks)
	waitGroup.Wait()
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

var nodeFinished map[*Node]struct{}

func printDepTreeArrows(line1, line2 *string, fieldstart, fieldlen int, midpoints ...int) (*string, *string) {
	for _, midpoint := range midpoints {
		// get the arrow direction (left, down, right)
		// 							/       |      \
		//                         V        V       V
		if midpoint < fieldstart { // left
			line1 = insertIntoString(line1, midpoint+2, "/")
			line2 = insertIntoString(line2, midpoint+1, "V")
		} else if midpoint < fieldstart+fieldlen { // down
			line1 = insertIntoString(line1, midpoint, "|")
			line2 = insertIntoString(line2, midpoint, "V")
		} else { // right
			line1 = insertIntoString(line1, midpoint-2, "\\")
			line2 = insertIntoString(line2, midpoint-1, "V")
		}
	}
	return line1, line2
}

func printDepTreeLevel(g *Graph, n *Node, out []*string, rightOffset int) (modout []*string, height int, width int) {
	dependencies := g.GetDependencies(n)
	name := " " + n.name + " "
	if _, finished := nodeFinished[n]; finished {
		name = " &" + n.name + " "
		dependencies = nil
	} else {
		nodeFinished[n] = struct{}{}
	}
	if len(dependencies) == 0 {
		if len(out) == 0 {
			line := strings.Repeat(" ", rightOffset) + name
			out = append(out, &line)
		} else {
			out[0] = insertIntoString(out[0], rightOffset, name)
		}
		// we are at the bottom of the graph
		return out, 1, len(name)
	}
	// add two lines for the arrows
	for len(out) < 3 {
		out = append(out, nil)
	}
	modout = out

	out = out[3:]
	// we have some dependencies, let us print them
	var depWidth, locHeight, locWidth int
	// save the midpoints to connect the arrows to
	depMids := make([]int, len(dependencies))
	for i, dep := range dependencies {
		out, locHeight, locWidth = printDepTreeLevel(g, dep, out, rightOffset+depWidth)
		depMids[i] = rightOffset + depWidth + locWidth/2
		depWidth += locWidth
		if height < locHeight {
			height = locHeight
		}
	}

	modout = append(modout[:3], out...)
	// print our name in the middle
	if len(name) > depWidth {
		// shift everything below to the right
		offsetLeft := (len(name) - depWidth) / 2
		offsetRight := (len(name) - depWidth + 1) / 2
		for _, strptr := range out {
			strptr = insertIntoString(insertIntoString(strptr, rightOffset+depWidth, strings.Repeat(" ", offsetRight)), rightOffset, strings.Repeat(" ", offsetLeft))
		}
		// same for the mid points
		for i, mid := range depMids {
			if mid < rightOffset+len(name)/2 {
				depMids[i] += offsetLeft
			} else {
				depMids[i] += offsetRight
			}
		}
		modout[1], modout[2] = printDepTreeArrows(modout[1], modout[2], rightOffset, len(name), depMids...)
	} else {
		modout[1], modout[2] = printDepTreeArrows(modout[1], modout[2], rightOffset+(depWidth-len(name))/2, len(name), depMids...)
		name = strings.Repeat(" ", (depWidth-len(name))/2) + name + strings.Repeat(" ", (depWidth-len(name)+1)/2)
	}
	modout[0] = insertIntoString(modout[0], rightOffset, name)
	return modout, height, len(name)
}

// printDepTree pretty prints the dependency tree of the specified startNode to stdout
func printDepTree(g *Graph, start *Node) {
	if start == nil {
		panic("printDepTree: start should not be nil!")
	}
	nodeFinished = map[*Node]struct{}{}
	out, _, _ := printDepTreeLevel(g, start, make([]*string, 1), 0)
	for _, lineptr := range out {
		println(*lineptr)
	}
}

// printFullDepTree prints the dependency tree of the whole graph to stdout
func printFullDepTree(g *Graph) {
	if len(g.nodes) == 0 {
		println("{empty graph}")
		return
	}
	// make a copy of our graph
	fullGraph := g.Copy()
	for _, node := range fullGraph.nodes {
		if len(fullGraph.GetDependants(node)) == 0 {
			fullGraph.AddEdge("_all", node.name)
		}
	}
	nodeFinished = map[*Node]struct{}{}
	out, _, _ := printDepTreeLevel(fullGraph, fullGraph.GetNode("_all"), make([]*string, 1), 0)
	for _, lineptr := range out[3:] {
		println(*lineptr)
	}
}
