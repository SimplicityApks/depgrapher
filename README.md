depgrapher   [![GoDoc](https://godoc.org/github.com/SimplicityApks/depgrapher?status.svg)](https://godoc.org/github.com/SimplicityApks/depgrapher)
==========

A simple command-line utility to generate dependency graphs from Makefiles written in Go.
Also contains a complete, ready-to-use graph implementation in [package graph](./graph).

Installation
------------

`go get github.com/SimplicityApks/depgrapher`

or clone repo and run
 
`go build`

Usage
-----

`depgrapher [-syntax syntaxname] [-node startname] [-outfile filename.dot|stdout] [file...]`

syntaxname has to be one of {Makefile, MakeCall, Dot} or a complete definition of a new syntax (see [package syntax](./syntax) 
for more information).

The optional startname restricts the output to the dependency graph of only the given node, instead of the whole graph.

If the outfile parameter is set, the graph will be printed in Graphviz dot syntax instead of a visual representation.
To get a nice graphical representation, you can pipe the output into Graphviz like so:   
`depgrapher -outfile stdout ... | dot -Tpng > picturename.png`

Example
-------

Suppose you have the following Makefile:

    all: build install test
    
    build: prepare compile pack

The command  
`depgrapher -syntax=Makefile Makefile`  
will produce the following output:

                      all                  
                  /           \       \
                 V             V       V
             build           install  test 
          /      |     \
         V       V      V
     prepare  compile  pack

License
-------

    This Source Code Form is subject to the terms of the Mozilla Public
    License, v. 2.0. If a copy of the MPL was not distributed with this
    file, You can obtain one at http://mozilla.org/MPL/2.0/.