// Contains utility functions for printing graphs
package main

import (
	"io"
	"strings"
)

// nodeFinished is a set that contains all the nodes which have already been printed. This is global for simplicity (& laziness)
var nodeFinished map[*Node]struct{}

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

func writeGraph(g *Graph, writer io.Writer, syntax Syntax) {
	writer.Write(append([]byte(syntax.GraphPrefix), '\n'))
	for _, node := range g.nodes {
		dependencies := g.GetDependencies(node)
		if len(dependencies) > 0 {
			writer.Write([]byte(syntax.EdgePrefix + node.String() + syntax.EdgeInfix))
			for index, dep := range dependencies {
				writer.Write([]byte(dep.String()))
				if index < len(dependencies)-1 {
					if syntax.TargetDelimiter == "" {
						writer.Write(append([]byte(syntax.EdgeSuffix + string('\n') + syntax.EdgePrefix + node.String() + syntax.EdgeInfix)))
					} else {
						writer.Write([]byte(syntax.TargetDelimiter))
					}
				}
			}
			writer.Write(append([]byte(syntax.EdgeSuffix), '\n'))
		}
	}
	writer.Write([]byte(syntax.GraphSuffix))
}

// writeDot writes the given graph to the given io.Writer in dot language syntax
func writeDot(g *Graph, writer io.Writer) {
	writeGraph(g, writer, DotSyntax)
}
