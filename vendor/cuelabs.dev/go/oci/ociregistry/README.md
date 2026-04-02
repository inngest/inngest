# `ociregistry`

In the top level package (`ociregistry`) this module defines a [Go interface](./interface.go) that encapsulates the operations provided by an OCI
registry.

Full reference documentation can be found [here](https://pkg.go.dev/cuelabs.dev/go/oci/ociregistry).

It also provides a lightweight in-memory implementation of that interface (`ocimem`)
and an HTTP server that implements the [OCI registry protocol](https://github.com/opencontainers/distribution-spec/blob/main/spec.md) on top of it.

The server currently passes the [conformance tests](https://pkg.go.dev/github.com/opencontainers/distribution-spec/conformance).

The aim is to provide an ergonomic interface for defining and layering
OCI registry implementations.

Although the API is fairly stable, it's still in v0 currently, so incompatible changes can't be ruled out.

The code was originally derived from the [go-containerregistry](https://pkg.go.dev/github.com/google/go-containerregistry/pkg/registry) registry, but has considerably diverged since then.
