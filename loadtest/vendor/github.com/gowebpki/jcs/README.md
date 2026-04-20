![JCS](https://cyberphone.github.io/doc/security/jcs.svg)

# JSON Canonicalization

[![Go Report Card](https://goreportcard.com/badge/github.com/gowebpki/jcs)](https://goreportcard.com/report/github.com/gowebpki/jcs) 
[![godoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://pkg.go.dev/github.com/gowebpki/jcs)
[![GitHub license](https://img.shields.io/github/license/gowebpki/jcs.svg?style=flat)](https://github.com/gowebpki/jcs/blob/master/LICENSE)
[![GitHub go.mod Go version of a Go module](https://img.shields.io/github/go-mod/go-version/gowebpki/jcs.svg?style=flat)](https://github.com/gowebpki/jcs)

Cryptographic operations like hashing and signing depend on that the target 
data does not change during serialization, transport, or parsing. 
By applying the rules defined by JCS (JSON Canonicalization Scheme), 
data provided in the JSON [[RFC8259](https://tools.ietf.org/html/rfc8259)]
format can be exchanged "as is", while still being subject to secure cryptographic operations.
JCS achieves this by building on the serialization formats for JSON
primitives as defined by ECMAScript [[ES](https://ecma-international.org/ecma-262/)],
constraining JSON data to the I-JSON [[RFC7493](https://tools.ietf.org/html//rfc7493)] subset,
and through a platform independent property sorting scheme.

Public RFC: https://tools.ietf.org/html/rfc8785

The JSON Canonicalization Scheme concept in a nutshell:
- Serialization of primitive JSON data types using methods compatible with ECMAScript's `JSON.stringify()`
- Lexicographic sorting of JSON `Object` properties in a *recursive* process
- JSON `Array` data is also subject to canonicalization, *but element order remains untouched*

### Original Work

This code was originally created by Anders Rundgren aka cyberphone and can be found here: 
https://github.com/cyberphone/json-canonicalization. This fork and work is done with Anders' 
permission and is an attempt to clean up the Golang version. 


### Sample Input
```code
{
  "numbers": [333333333.33333329, 1E30, 4.50, 2e-3, 0.000000000000000000000000001],
  "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/",
  "literals": [null, true, false]
}
```
### Expected Output
```code
{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],"string":"€$\u000f\nA'B\"\\\\\"/"}
```

Note: for platform interoperable canonicalization, the output must be converted to UTF-8
as well, here shown in hexadecimal notation:

```code
7b 22 6c 69 74 65 72 61 6c 73 22 3a 5b 6e 75 6c 6c 2c 74 72 75 65 2c 66 61 6c 73 65 5d 2c 22 6e
75 6d 62 65 72 73 22 3a 5b 33 33 33 33 33 33 33 33 33 2e 33 33 33 33 33 33 33 2c 31 65 2b 33 30
2c 34 2e 35 2c 30 2e 30 30 32 2c 31 65 2d 32 37 5d 2c 22 73 74 72 69 6e 67 22 3a 22 e2 82 ac 24
5c 75 30 30 30 66 5c 6e 41 27 42 5c 22 5c 5c 5c 5c 5c 22 2f 22 7d
```
### Combining JCS and JWS (RFC7515)
[JWS-JCS](https://github.com/cyberphone/jws-jcs#combining-detached-jws-with-jcs-json-canonicalization-scheme)

### On-line Browser JCS Test
https://cyberphone.github.io/doc/security/browser-json-canonicalization.html

### ECMAScript Proposal: JSON.canonify()
[JSON.canonify()](https://github.com/cyberphone/json-canonicalization/blob/master/JSON.canonify.md)

### Other Canonicalization Efforts
https://tools.ietf.org/html/draft-staykov-hu-json-canonical-form-00

http://wiki.laptop.org/go/Canonical_JSON

https://gibson042.github.io/canonicaljson-spec/

https://gist.github.com/mikesamuel/20710f94a53e440691f04bf79bc3d756
