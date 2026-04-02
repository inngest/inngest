// Copyright 2023 CUE Labs AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ociref supports parsing cross-registry OCI registry references.
package ociref

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/opencontainers/go-digest"
)

// The following regular expressions derived from code in the
// [github.com/distribution/distribution/v3/reference] package.
const (
	// alphanumeric defines the alphanumeric atom, typically a
	// component of names. This only allows lower case characters and digits.
	alphanumeric = `[a-z0-9]+`

	// separator defines the separators allowed to be embedded in name
	// components. This allows one period, one or two underscore and multiple
	// dashes. Repeated dashes and underscores are intentionally treated
	// differently. In order to support valid hostnames as name components,
	// supporting repeated dash was added. Additionally double underscore is
	// now allowed as a separator to loosen the restriction for previously
	// supported names.
	// TODO the distribution spec doesn't allow these variations.
	separator = `(?:[._]|__|[-]+)`

	// domainNameComponent restricts the registry domain component of a
	// repository name to start with a component as defined by DomainRegexp.
	domainNameComponent = `(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)`

	// ipv6address are enclosed between square brackets and may be represented
	// in many ways, see rfc5952. Only IPv6 in compressed or uncompressed format
	// are allowed, IPv6 zone identifiers (rfc6874) or Special addresses such as
	// IPv4-Mapped are deliberately excluded.
	ipv6address = `(?:\[[a-fA-F0-9:]+\])`

	// optionalPort matches an optional port-number including the port separator
	// (e.g. ":80").
	port = `[0-9]+`

	// domainName defines the structure of potential domain components
	// that may be part of image names. This is purposely a subset of what is
	// allowed by DNS to ensure backwards compatibility with Docker image
	// names. This includes IPv4 addresses on decimal format.
	//
	// Note: we purposely exclude domain names without dots here,
	// because otherwise we can't tell if the first component is
	// a host name or not when it doesn't have a port.
	// When it does have a port, the distinction is clear.
	//
	domainName = `(?:` + domainNameComponent + `(?:\.` + domainNameComponent + `)+` + `)`

	// host defines the structure of potential domains based on the URI
	// Host subcomponent on rfc3986. It may be a subset of DNS domain name,
	// or an IPv4 address in decimal format, or an IPv6 address between square
	// brackets (excluding zone identifiers as defined by rfc6874 or special
	// addresses such as IPv4-Mapped).
	host = `(?:` + domainName + `|` + ipv6address + `)`

	// allowed by the URI Host subcomponent on rfc3986 to ensure backwards
	// compatibility with Docker image names.
	// Note: that we require the port when the host name looks like a regular
	// name component.
	domainAndPort = `(?:` + host + `(?:` + `:` + port + `)?` + `|` + domainNameComponent + `:` + port + `)`

	// pathComponent restricts path-components to start with an alphanumeric
	// character, with following parts able to be separated by a separator
	// (one period, one or two underscore and multiple dashes).
	pathComponent = `(?:` + alphanumeric + `(?:` + separator + alphanumeric + `)*` + `)`

	// repoName matches the name of a repository. It consists of one
	// or more forward slash (/) delimited path-components:
	//
	//	pathComponent[[/pathComponent] ...] // e.g., "library/ubuntu"
	repoName = pathComponent + `(?:` + `/` + pathComponent + `)*`
)

var referencePat = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(
		`^(?:` +
			`(?:` + `(` + domainAndPort + `)` + `/` + `)?` + // capture 1: host
			`(` + repoName + `)` + // capture 2: repository name
			`(?:` + `:([^@]+))?` + // capture 3: tag; rely on Go logic to test validity.
			`(?:` + `@(.+))?` + // capture 4: digest; rely on go-digest to find issues
			`)$`,
	)
})

var hostPat = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^(?:` + domainAndPort + `)$`)
})
var repoPat = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^(?:` + repoName + `)$`)
})

// Reference represents an entry in an OCI repository.
type Reference struct {
	// Host holds the host name of the registry
	// within which the repository is stored, optionally in
	// the form host:port. This might be empty.
	Host string

	// Repository holds the repository name.
	Repository string

	// Tag holds the TAG part of a :TAG or :TAG@DIGEST reference.
	// When Digest is set as well as Tag, the tag will be verified
	// to exist and have the expected digest.
	Tag string

	// Digest holds the DIGEST part of an @DIGEST reference
	// or of a :TAG@DIGEST reference.
	Digest Digest
}

type Digest = digest.Digest

// IsValidHost reports whether s is a valid host (or host:port) part of a reference string.
func IsValidHost(s string) bool {
	return hostPat().MatchString(s)
}

// IsValidHost reports whether s is a valid repository part
// of a reference string.
func IsValidRepository(s string) bool {
	return repoPat().MatchString(s)
}

// IsValidTag reports whether s is a valid reference tag.
func IsValidTag(s string) bool {
	return checkTag(s) == nil
}

// IsValidDigest reports whether the digest d is well formed.
func IsValidDigest(d string) bool {
	_, err := digest.Parse(d)
	return err == nil
}

// Parse parses a reference string that must include
// a host name (or host:port pair) component.
//
// It is represented in string form as HOST[:PORT]/NAME[:TAG|@DIGEST]
// form: the same syntax accepted by "docker pull".
// Unlike "docker pull" however, there is no default registry: when
// presented with a bare repository name, Parse will return an error.
func Parse(refStr string) (Reference, error) {
	ref, err := ParseRelative(refStr)
	if err != nil {
		return Reference{}, err
	}
	if ref.Host == "" {
		return Reference{}, fmt.Errorf("reference does not contain host name")
	}
	return ref, nil
}

// ParseRelative parses a reference string that may
// or may not include a host name component.
//
// It is represented in string form as [HOST[:PORT]/]NAME[:TAG|@DIGEST]
// form: the same syntax accepted by "docker pull".
// Unlike "docker pull" however, there is no default registry: when
// presented with a bare repository name, the Host field will be empty.
func ParseRelative(refStr string) (Reference, error) {
	m := referencePat().FindStringSubmatch(refStr)
	if m == nil {
		return Reference{}, fmt.Errorf("invalid reference syntax (%q)", refStr)
	}
	var ref Reference
	ref.Host, ref.Repository, ref.Tag, ref.Digest = m[1], m[2], m[3], Digest(m[4])
	// Check lengths and digest: we don't check these as part of the regexp
	// because it's more efficient to do it in Go and we get
	// nicer error messages as a result.
	if len(ref.Digest) > 0 {
		if err := ref.Digest.Validate(); err != nil {
			return Reference{}, fmt.Errorf("invalid digest %q: %v", ref.Digest, err)
		}
	}
	if len(ref.Tag) > 0 {
		if err := checkTag(ref.Tag); err != nil {
			return Reference{}, err
		}
	}
	if len(ref.Repository) > 255 {
		return Reference{}, fmt.Errorf("repository name too long")
	}
	return ref, nil
}

func checkTag(s string) error {
	if len(s) > 128 {
		return fmt.Errorf("tag too long")
	}
	if !isWord(s[0]) {
		return fmt.Errorf("tag %q does not start with word character", s)
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !isWord(c) && c != '.' && c != '-' {
			return fmt.Errorf("tag %q contains invalid invalid character %q", s, c)
		}
	}
	return nil
}

func isWord(c byte) bool {
	return c == '_' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')
}

// String returns the string form of a reference in the form
//
//	[HOST/]NAME[:TAG|@DIGEST]
func (ref Reference) String() string {
	var buf strings.Builder
	buf.Grow(len(ref.Host) + 1 + len(ref.Repository) + 1 + len(ref.Tag) + 1 + len(ref.Digest))
	if ref.Host != "" {
		buf.WriteString(ref.Host)
		buf.WriteByte('/')
	}
	buf.WriteString(ref.Repository)
	if len(ref.Tag) > 0 {
		buf.WriteByte(':')
		buf.WriteString(ref.Tag)
	}
	if len(ref.Digest) > 0 {
		buf.WriteByte('@')
		buf.WriteString(string(ref.Digest))
	}
	return buf.String()
}
