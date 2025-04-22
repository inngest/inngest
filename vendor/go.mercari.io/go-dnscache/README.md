# go-dnscache [![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godoc] [![CircleCI (all branches)](https://img.shields.io/circleci/project/github/mercari/go-dnscache.svg?style=flat-square)](circleci)

[godoc]: http://godoc.org/go.mercari.io/go-dnscache
[circleci]: https://circleci.com/gh/mercari/go-dnscache

`go-dnscache` is a Go package for caching DNS lookup results in memory. It asynchronously lookups DNS and refresh results. The main motivation of this package is to avoid too much DNS lookups for every request (DNS lookup sometimes makes request really slow and causes error). This can be mainly used for the targets which are running on *non-dynamic* environment where IP does not change often.

## Install

Use go get:

```bash
$ go get -u go.mercari.io/go-dnscache
```

## Usage

All usage are described in [GoDoc](https://godoc.org/go.mercari.io/go-dnscache).
