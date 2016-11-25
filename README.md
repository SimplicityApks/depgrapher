depgrapher
==========

A simple command-line utility to generate dependency graphs from Makefiles written in Go.

Installation
------------

`go get github.com/SimplicityApks/depgrapher`

or checkout repo and run
 
`go build`

Usage
-----

`depgrapher [-syntax syntaxname] [-outfile filename|stdout] [file...]`

Example:

`depgrapher -syntax=Makefile -outfile stdout Makefile`

License
-------

    This Source Code Form is subject to the terms of the Mozilla Public
    License, v. 2.0. If a copy of the MPL was not distributed with this
    file, You can obtain one at http://mozilla.org/MPL/2.0/.