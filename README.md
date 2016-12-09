depgrapher   [![GoDoc](https://godoc.org/github.com/SimplicityApks/depgrapher?status.svg)](https://godoc.org/github.com/SimplicityApks/depgrapher)
==========

A simple command-line utility to generate dependency graphs from Makefiles written in Go.

Installation
------------

`go get github.com/SimplicityApks/depgrapher`

or clone repo and run
 
`go build`

Usage
-----

`depgrapher [-syntax syntaxname] [-node startname] [-outfile filename.dot|stdout] [file...]`

Example:

`depgrapher -syntax=Makefile -node all -outfile stdout Makefile`

License
-------

    This Source Code Form is subject to the terms of the Mozilla Public
    License, v. 2.0. If a copy of the MPL was not distributed with this
    file, You can obtain one at http://mozilla.org/MPL/2.0/.