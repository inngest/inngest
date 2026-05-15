// Package types provides collected type definitions for TypeScript generation.
//
// This package contains types collected from multiple packages by tygo-collect,
// allowing tygo to generate TypeScript from a single source.
//
// To regenerate TypeScript types:
//
//	go generate ./pkg/tracing/metadata/types
//	tygo generate
package types

//go:generate go run ../../../../cmd/tygo-collect -o types_gen.go ../extractors ..
