// Copyright 2021 Bret Jordan & Benedikt Thoma, All rights reserved.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// Use of this source code is governed by an Apache 2.0 license that can be
// found in the LICENSE file in the root of the source tree.

// Package jcs transforms UTF-8 JSON data into a canonicalized version according RFC 8785
package jcs

import (
	"container/list"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

type nameValueType struct {
	name    string
	sortKey []uint16
	value   string
}

type jcsData struct {
	// JSON data MUST be UTF-8 encoded
	jsonData []byte
	// Current pointer in jsonData
	index int
}

// JSON standard escapes (modulo \u)
var (
	asciiEscapes  = []byte{'\\', '"', 'b', 'f', 'n', 'r', 't'}
	binaryEscapes = []byte{'\\', '"', '\b', '\f', '\n', '\r', '\t'}
)

// JSON literals
var literals = []string{"true", "false", "null"}

// Transform converts raw JSON data from a []byte array into a canonicalized version according RFC 8785
func Transform(jsonData []byte) ([]byte, error) {
	if jsonData == nil {
		return nil, errors.New("No JSON data provided")
	}

	// Create a JCS Data struct to store the JSON Data and the index.
	var jd jcsData
	jd.jsonData = jsonData
	j := &jd

	transformed, err := j.parseEntry()
	if err != nil {
		return nil, err
	}

	for j.index < len(j.jsonData) {
		if !j.isWhiteSpace(j.jsonData[j.index]) {
			return nil, errors.New("Improperly terminated JSON object")
		}
		j.index++
	}
	return []byte(transformed), err
}

func (j *jcsData) isWhiteSpace(c byte) bool {
	return c == 0x20 || c == 0x0a || c == 0x0d || c == 0x09
}

func (j *jcsData) nextChar() (byte, error) {
	if j.index < len(j.jsonData) {
		c := j.jsonData[j.index]
		if c > 0x7f {
			return 0, errors.New("Unexpected non-ASCII character")
		}
		j.index++
		return c, nil
	}
	return 0, errors.New("Unexpected EOF reached")
}

// scan advances index on jsonData to the first non whitespace character and returns it.
func (j *jcsData) scan() (byte, error) {
	for {
		c, err := j.nextChar()
		if err != nil {
			return 0, err
		}

		if j.isWhiteSpace(c) {
			continue
		}

		return c, nil
	}
}

func (j *jcsData) scanFor(expected byte) error {
	c, err := j.scan()
	if err != nil {
		return err
	}
	if c != expected {
		return fmt.Errorf("Expected %s but got %s", string(expected), string(c))
	}
	return nil
}

func (j *jcsData) getUEscape() (rune, error) {
	start := j.index
	for i := 0; i < 4; i++ {
		_, err := j.nextChar()
		if err != nil {
			return 0, err
		}
	}

	u16, err := strconv.ParseUint(string(j.jsonData[start:j.index]), 16, 64)
	if err != nil {
		return 0, err
	}
	return rune(u16), nil
}

func (j *jcsData) decorateString(rawUTF8 string) string {
	var quotedString strings.Builder
	quotedString.WriteByte('"')

CoreLoop:
	for _, c := range []byte(rawUTF8) {
		// Is this within the JSON standard escapes?
		for i, esc := range binaryEscapes {
			if esc == c {
				quotedString.WriteByte('\\')
				quotedString.WriteByte(asciiEscapes[i])

				continue CoreLoop
			}
		}
		if c < 0x20 {
			// Other ASCII control characters must be escaped with \uhhhh
			quotedString.WriteString(fmt.Sprintf("\\u%04x", c))
		} else {
			quotedString.WriteByte(c)
		}
	}
	quotedString.WriteByte('"')

	return quotedString.String()
}

// parseEntry is the entrypoint into the parsing control flow
func (j *jcsData) parseEntry() (string, error) {
	c, err := j.scan()
	if err != nil {
		return "", err
	}
	j.index--

	switch c {
	case '{', '"', '[':
		return j.parseElement()
	default:
		value, err := parseLiteral(string(j.jsonData))
		if err != nil {
			return "", err
		}

		j.index = len(j.jsonData)
		return value, nil
	}
}

func (j *jcsData) parseQuotedString() (string, error) {
	var rawString strings.Builder

CoreLoop:
	for {
		var c byte
		if j.index < len(j.jsonData) {
			c = j.jsonData[j.index]
			j.index++
		} else {
			return "", errors.New("Unexpected EOF reached")
		}

		if c == '"' {
			break
		}

		if c < ' ' {
			return "", errors.New("Unterminated string literal")
		} else if c == '\\' {
			// Escape sequence
			c, err := j.nextChar()
			if err != nil {
				return "", err
			}

			if c == 'u' {
				// The \u escape
				firstUTF16, err := j.getUEscape()
				if err != nil {
					return "", err
				}

				if utf16.IsSurrogate(firstUTF16) {
					// If the first UTF-16 code unit has a certain value there must be
					// another succeeding UTF-16 code unit as well
					backslash, err := j.nextChar()
					if err != nil {
						return "", err
					}
					u, err := j.nextChar()
					if err != nil {
						return "", err
					}

					if backslash != '\\' || u != 'u' {
						return "", errors.New("Missing surrogate")
					}

					// Output the UTF-32 code point as UTF-8
					uEscape, err := j.getUEscape()
					if err != nil {
						return "", err
					}
					rawString.WriteRune(utf16.DecodeRune(firstUTF16, uEscape))

				} else {
					// Single UTF-16 code identical to UTF-32.  Output as UTF-8
					rawString.WriteRune(firstUTF16)
				}
			} else if c == '/' {
				// Benign but useless escape
				rawString.WriteByte('/')
			} else {
				// The JSON standard escapes
				for i, esc := range asciiEscapes {
					if esc == c {
						rawString.WriteByte(binaryEscapes[i])
						continue CoreLoop
					}
				}
				return "", fmt.Errorf("Unexpected escape: \\%s", string(c))
			}
		} else {
			// Just an ordinary ASCII character alternatively a UTF-8 byte
			// outside of ASCII.
			// Note that properly formatted UTF-8 never clashes with ASCII
			// making byte per byte search for ASCII break characters work
			// as expected.
			rawString.WriteByte(c)
		}
	}

	return rawString.String(), nil
}

func (j *jcsData) parseSimpleType() (string, error) {
	var token strings.Builder

	j.index--

	// no condition is needed here.
	// if the buffer reaches EOF scan returns an error, or we terminate because the
	// json simple type terminates
	for {
		c, err := j.scan()
		if err != nil {
			return "", err
		}

		if c == ',' || c == ']' || c == '}' {
			j.index--
			break
		}

		token.WriteByte(c)
	}

	if token.Len() == 0 {
		return "", errors.New("Missing argument")
	}

	return parseLiteral(token.String())
}

func parseLiteral(value string) (string, error) {
	// Is it a JSON literal?
	for _, literal := range literals {
		if literal == value {
			return literal, nil
		}
	}

	// Apparently not so we assume that it is a I-JSON number
	ieeeF64, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", err
	}

	value, err = NumberToJSON(ieeeF64)
	if err != nil {
		return "", err
	}

	return value, nil
}

func (j *jcsData) parseElement() (string, error) {
	c, err := j.scan()
	if err != nil {
		return "", err
	}

	switch c {
	case '{':
		return j.parseObject()
	case '"':
		str, err := j.parseQuotedString()
		if err != nil {
			return "", err
		}
		return j.decorateString(str), nil
	case '[':
		return j.parseArray()
	default:
		return j.parseSimpleType()
	}
}

func (j *jcsData) peek() (byte, error) {
	c, err := j.scan()
	if err != nil {
		return 0, err
	}

	j.index--
	return c, nil
}

func (j *jcsData) parseArray() (string, error) {
	var arrayData strings.Builder
	var next bool

	arrayData.WriteByte('[')

	for {
		c, err := j.peek()
		if err != nil {
			return "", err
		}

		if c == ']' {
			j.index++
			break
		}

		if next {
			err = j.scanFor(',')
			if err != nil {
				return "", err
			}
			arrayData.WriteByte(',')
		} else {
			next = true
		}

		element, err := j.parseElement()
		if err != nil {
			return "", err
		}
		arrayData.WriteString(element)
	}

	arrayData.WriteByte(']')
	return arrayData.String(), nil
}

func (j *jcsData) lexicographicallyPrecedes(sortKey []uint16, e *list.Element) (bool, error) {
	// Find the minimum length of the sortKeys
	oldSortKey := e.Value.(nameValueType).sortKey
	minLength := len(oldSortKey)
	if minLength > len(sortKey) {
		minLength = len(sortKey)
	}
	for q := 0; q < minLength; q++ {
		diff := int(sortKey[q]) - int(oldSortKey[q])
		if diff < 0 {
			// Smaller => Precedes
			return true, nil
		} else if diff > 0 {
			// Bigger => No match
			return false, nil
		}
		// Still equal => Continue
	}
	// The sortKeys compared equal up to minLength
	if len(sortKey) < len(oldSortKey) {
		// Shorter => Precedes
		return true, nil
	}
	if len(sortKey) == len(oldSortKey) {
		return false, fmt.Errorf("Duplicate key: %s", e.Value.(nameValueType).name)
	}
	// Longer => No match
	return false, nil
}

func (j *jcsData) parseObject() (string, error) {
	nameValueList := list.New()
	var next bool = false
CoreLoop:
	for {
		c, err := j.peek()
		if err != nil {
			return "", err
		}

		if c == '}' {
			// advance index because of peeked '}'
			j.index++
			break
		}

		if next {
			err = j.scanFor(',')
			if err != nil {
				return "", err
			}
		}
		next = true

		err = j.scanFor('"')
		if err != nil {
			return "", err
		}
		rawUTF8, err := j.parseQuotedString()
		if err != nil {
			break
		}
		// Sort keys on UTF-16 code units
		// Since UTF-8 doesn't have endianess this is just a value transformation
		// In the Go case the transformation is UTF-8 => UTF-32 => UTF-16
		sortKey := utf16.Encode([]rune(rawUTF8))
		err = j.scanFor(':')
		if err != nil {
			return "", err
		}

		element, err := j.parseElement()
		if err != nil {
			return "", err
		}
		nameValue := nameValueType{rawUTF8, sortKey, element}
		for e := nameValueList.Front(); e != nil; e = e.Next() {
			// Check if the key is smaller than a previous key
			if precedes, err := j.lexicographicallyPrecedes(sortKey, e); err != nil {
				return "", err
			} else if precedes {
				// Precedes => Insert before and exit sorting
				nameValueList.InsertBefore(nameValue, e)
				continue CoreLoop
			}
			// Continue searching for a possibly succeeding sortKey
			// (which is straightforward since the list is ordered)
		}
		// The sortKey is either the first or is succeeding all previous sortKeys
		nameValueList.PushBack(nameValue)
	}

	// Now everything is sorted so we can properly serialize the object
	var objectData strings.Builder
	objectData.WriteByte('{')
	next = false
	for e := nameValueList.Front(); e != nil; e = e.Next() {
		if next {
			objectData.WriteByte(',')
		}
		next = true
		nameValue := e.Value.(nameValueType)
		objectData.WriteString(j.decorateString(nameValue.name))
		objectData.WriteByte(':')
		objectData.WriteString(nameValue.value)
	}
	objectData.WriteByte('}')
	return objectData.String(), nil
}
