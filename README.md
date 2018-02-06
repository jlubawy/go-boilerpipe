# go-boilerpipe

_Golang port of the [boilerpipe](https://github.com/kohlschutter/boilerpipe)
Java library written by Christian Kohlschütter_

[![Build Status](https://travis-ci.org/jlubawy/go-boilerpipe.svg?branch=master)](https://travis-ci.org/jlubawy/go-boilerpipe) [![GoDoc](https://godoc.org/github.com/jlubawy/go-boilerpipe?status.svg)](https://godoc.org/github.com/jlubawy/go-boilerpipe)

Boilerpipe removes boilerplate and extracts text content from HTML documents.

Currently it only supports article extraction which includes the title,
the date, and the content.

Best attempts will be made to follow [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html#semantic-versioning-200) rules,
but no API guarantees will be made until version 1.0.0.

However, existing tags will not change guaranteeing that [vendoring](https://semver.org/spec/v2.0.0.html#semantic-versioning-200) will not break.


## Getting Started

To install: ```go get -u github.com/jlubawy/go-boilerpipe/...```


## Command-Line Tool

    $ boilerpipe help

    Boilerpipe removes boilerplate and extracts text content from HTML documents.

    Usage:

           boilerpipe command [arguments]

    The commands are:

           extract    extract text content from an HTML document
           serve      start a HTTP server for extracting text from HTML documents
           version    print boilerpipe version

    Use "boilerpipe help [command]" for more information about a command.


## Using the library

See examples in [filter_test.go](filter_test.go).
