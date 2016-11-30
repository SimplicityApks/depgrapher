/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

// Contains utility functions for printing graphs that work with every implementation of graph.Interface.
package graph

import (
	"github.com/SimplicityApks/depgrapher/syntax"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// nodeFinished is a set that contains all the nodes which have already been printed. This is global for simplicity (& laziness).
var nodeFinished map[Node]struct{}

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

func printDepTreeLevel(g Interface, n Node, out []*string, rightOffset int) (modout []*string, width int) {
	dependencies := g.GetDependencies(n.String())
	name := " " + n.String() + " "
	if _, finished := nodeFinished[n]; finished {
		name = " &" + n.String() + " "
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
		return out, len(name)
	}
	// add two lines for the arrows
	for len(out) < 3 {
		out = append(out, nil)
	}
	modout = out

	out = out[3:]
	// we have some dependencies, let us print them
	var depWidth, locWidth int
	// save the midpoints to connect the arrows to
	depMids := make([]int, len(dependencies))
	for i, dep := range dependencies {
		out, locWidth = printDepTreeLevel(g, dep, out, rightOffset+depWidth)
		depMids[i] = rightOffset + depWidth + locWidth/2
		depWidth += locWidth
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
	return modout, len(name)
}

// PrintDepTree pretty prints the dependency tree of the specified startNode to stdout.
func PrintDepTree(graph Interface, start Node) {
	if start == nil {
		panic("printDepTree: start should not be nil!")
	}
	nodeFinished = map[Node]struct{}{}
	out := wrapToStdout(printDepTreeLevel(graph, start, make([]*string, 1), 0))
	for _, lineptr := range out {
		println(*lineptr)
	}
}

// PrintFullDepTree prints the dependency tree of the whole graph to stdout.
func PrintFullDepTree(graph Interface) {
	if len(graph.GetNodes()) == 0 {
		println("{empty graph}")
		return
	}
	// make a copy of our graph
	fullGraph := graph.Copy()
	for _, n := range fullGraph.GetNodes() {
		if len(fullGraph.GetDependants(n.String())) == 0 {
			fullGraph.AddEdge(node("_all"), n)
		}
	}
	nodeFinished = map[Node]struct{}{}
	out, width := printDepTreeLevel(fullGraph, fullGraph.GetNode("_all"), make([]*string, 1), 0)
	out = wrapToStdout(out[3:], width)
	for _, lineptr := range out {
		println(*lineptr)
	}
}

// wrapToStdout wraps the given output array to the size of os.Stdout.
func wrapToStdout(out []*string, width int) []*string {
	// wrap the lines if our output buffer is too small for the width of the full graph
	stdoutWidth, err := getStdoutWidth()
	if err != nil {
		panic("printDepTree: unable to get size of stdout: " + err.Error())
	}
	if stdoutWidth <= 0 { // some terminals have approximately infinite buffer size
		println("printDepTree: size of stdout is zero!")
		return out
	}
	startHeight := 0
	for width > stdoutWidth {
		emptyLine := ""
		out = append(out, &emptyLine)
		height := len(out)
		// TODO: think of a nicer wrapping method
		for index := startHeight; index < height; index++ {
			line := *out[index]
			if len(line) > stdoutWidth {
				part1, part2 := line[:stdoutWidth], line[stdoutWidth:]
				out[index] = &part1
				out = append(out, &part2)
			}
		}
		startHeight = height
		width -= stdoutWidth
	}
	return out
}

// getStdoutWidth returns the width of the stdout window.
func getStdoutWidth() (width int, err error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return -1, err
	}
	widthstring := strings.Split(string(out), " ")[1]
	return strconv.Atoi(widthstring[:len(widthstring)-1])
}

// WriteGraph writes a machine-readable version of the graph to writer, matching the given syntax.
func WriteGraph(graph Interface, writer io.Writer, syntax *syntax.Syntax) {
	writer.Write(append([]byte(syntax.GraphPrefix), '\n'))
	for _, node := range graph.GetNodes() {
		dependencies := graph.GetDependencies(node.String())
		if len(dependencies) > 0 {
			writer.Write([]byte(syntax.EdgePrefix + "\"" + node.String() + "\"" + syntax.EdgeInfix))
			for index, dep := range dependencies {
				writer.Write([]byte("\"" + dep.String() + "\""))
				if index < len(dependencies)-1 {
					if syntax.TargetDelimiter == "" {
						writer.Write(append([]byte(syntax.EdgeSuffix + string('\n') + syntax.EdgePrefix + "\"" + node.String() + "\"" + syntax.EdgeInfix)))
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

// WriteDot writes the given graph to the given io.Writer in dot language syntax.
func WriteDot(graph Interface, writer io.Writer) {
	WriteGraph(graph, writer, syntax.Dot)
}
