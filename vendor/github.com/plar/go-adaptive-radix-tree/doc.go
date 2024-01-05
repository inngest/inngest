// Package art implements an Adapative Radix Tree(ART) in pure Go.
// Note that this implementation is not thread-safe but it could be really easy to implement.
//
// The design of ART is based on "The Adaptive Radix Tree: ARTful Indexing for Main-Memory Databases" [1].
//
// Usage
//
//  package main
//
//  import (
//     "fmt"
//     "github.com/plar/go-adaptive-radix-tree"
//  )
//
//  func main() {
//
//     tree := art.New()
//
//     tree.Insert(art.Key("Hi, I'm Key"), "Nice to meet you, I'm Value")
//     value, found := tree.Search(art.Key("Hi, I'm Key"))
//     if found {
//         fmt.Printf("Search value=%v\n", value)
//     }
//
//     tree.ForEach(func(node art.Node) bool {
//         fmt.Printf("Callback value=%v\n", node.Value())
//         return true
//     }
//
//     for it := tree.Iterator(); it.HasNext(); {
//         value, _ := it.Next()
//         fmt.Printf("Iterator value=%v\n", value.Value())
//     }
//  }
//
//  // Output:
//  // Search value=Nice to meet you, I'm Value
//  // Callback value=Nice to meet you, I'm Value
//  // Iterator value=Nice to meet you, I'm Value
//
//
// Also the current implementation was inspired by [2] and [3]
//
// [1] http://db.in.tum.de/~leis/papers/ART.pdf (Specification)
//
// [2] https://github.com/armon/libart (C99 implementation)
//
// [3] https://github.com/kellydunn/go-art (other Go implementation)
package art
