// Copyright (c) 2020, Peter Ohler, All rights reserved.

package ojg

import (
	"fmt"
	"strconv"
	"time"
)

const (
	// Normal is the Normal ANSI encoding sequence.
	Normal = "\x1b[m"
	// Black is the Black ANSI encoding sequence.
	Black = "\x1b[30m"
	// Red is the Red ANSI encoding sequence.
	Red = "\x1b[31m"
	// Green is the Green ANSI encoding sequence.
	Green = "\x1b[32m"
	// Yellow is the Yellow ANSI encoding sequence.
	Yellow = "\x1b[33m"
	// Blue is the Blue ANSI encoding sequence.
	Blue = "\x1b[34m"
	// Magenta is the Magenta ANSI encoding sequence.
	Magenta = "\x1b[35m"
	// Cyan is the Cyan ANSI encoding sequence.
	Cyan = "\x1b[36m"
	// White is the White ANSI encoding sequence.
	White = "\x1b[37m"
	// Gray is the Gray ANSI encoding sequence.
	Gray = "\x1b[90m"
	// BrightRed is the BrightRed ANSI encoding sequence.
	BrightRed = "\x1b[91m"
	// BrightGreen is the BrightGreen ANSI encoding sequence.
	BrightGreen = "\x1b[92m"
	// BrightYellow is the BrightYellow ANSI encoding sequence.
	BrightYellow = "\x1b[93m"
	// BrightBlue is the BrightBlue ANSI encoding sequence.
	BrightBlue = "\x1b[94m"
	// BrightMagenta is the BrightMagenta ANSI encoding sequence.
	BrightMagenta = "\x1b[95m"
	// BrightCyan is the BrightCyan ANSI encoding sequence.
	BrightCyan = "\x1b[96m"
	// BrightWhite is the BrightWhite ANSI encoding sequence.
	BrightWhite = "\x1b[97m"

	// BytesAsString indicates []byte should be encoded as a string.
	BytesAsString = iota
	// BytesAsBase64 indicates []byte should be encoded as base64.
	BytesAsBase64
	// BytesAsArray indicates []byte should be encoded as an array if integers.
	BytesAsArray

	// MaskByTag is the mask for byTag fields.
	MaskByTag = byte(0x10)
	// MaskExact is the mask for Exact fields.
	MaskExact = byte(0x08) // exact key vs lowwer case first letter
	// MaskPretty is the mask for Pretty fields.
	MaskPretty = byte(0x04)
	// MaskNested is the mask for Nested fields.
	MaskNested = byte(0x02)
	// MaskSen is the mask for Sen fields.
	MaskSen = byte(0x01)
	// MaskSet is the mask for Set fields.
	MaskSet = byte(0x20)
	// MaskIndex is the mask for an index that has been set up.
	MaskIndex = byte(0x1f)
)

var (
	// DefaultOptions default options that can be set as desired.
	DefaultOptions = Options{
		InitSize:    256,
		SyntaxColor: Normal,
		KeyColor:    Blue,
		NullColor:   Red,
		BoolColor:   Yellow,
		NumberColor: Cyan,
		StringColor: Green,
		TimeColor:   Magenta,
		HTMLUnsafe:  true,
		WriteLimit:  1024,
	}

	// BrightOptions encoding options for color encoding.
	BrightOptions = Options{
		InitSize:    256,
		SyntaxColor: Normal,
		KeyColor:    BrightBlue,
		NullColor:   BrightRed,
		BoolColor:   BrightYellow,
		NumberColor: BrightCyan,
		StringColor: BrightGreen,
		TimeColor:   BrightMagenta,
		WriteLimit:  1024,
	}

	// GoOptions are the options closest to the go json package.
	GoOptions = Options{
		InitSize:     256,
		SyntaxColor:  Normal,
		KeyColor:     Blue,
		NullColor:    Red,
		BoolColor:    Yellow,
		NumberColor:  Cyan,
		StringColor:  Green,
		TimeColor:    Magenta,
		CreateKey:    "",
		FullTypePath: false,
		OmitNil:      false,
		OmitEmpty:    false,
		UseTags:      true,
		KeyExact:     true,
		NestEmbed:    false,
		BytesAs:      BytesAsBase64,
		WriteLimit:   1024,
	}

	// HTMLOptions defines color options for generating colored HTML. The
	// encoding is suitable for use in a <pre> element.
	HTMLOptions = Options{
		InitSize:    256,
		SyntaxColor: "<span>",
		KeyColor:    `<span style="color:#44f">`,
		NullColor:   `<span style="color:red">`,
		BoolColor:   `<span style="color:#a40">`,
		NumberColor: `<span style="color:#04a">`,
		StringColor: `<span style="color:green">`,
		TimeColor:   `<span style="color:#f0f">`,
		NoColor:     "</span>",
		HTMLUnsafe:  false,
		WriteLimit:  1024,
	}
)

// Options for writing data to JSON.
type Options struct {

	// Indent for the output.
	Indent int

	// Tab if true will indent using tabs and ignore the Indent member.
	Tab bool

	// Sort object members if true.
	Sort bool

	// OmitNil skips the writing of nil values in an object.
	OmitNil bool

	// OmitEmpty skips the writing of empty string, slices, maps, and zero
	// values although maps with all empty members will not be skipped on
	// writing but will be with alt.Decompose and alter.
	OmitEmpty bool

	// InitSize is the initial buffer size.
	InitSize int

	// WriteLimit is the size of the buffer that will trigger a write when
	// using a writer.
	WriteLimit int

	// TimeFormat defines how time is encoded. Options are to use a
	// time. layout string format such as time.RFC3339Nano, "second" for a
	// decimal representation, "nano" for a an integer. For decompose setting
	// to "time" will leave it unchanged.
	TimeFormat string

	// TimeWrap if not empty encoded time as an object with a single member. For
	// example if set to "@" then and TimeFormat is RFC3339Nano then the encoded
	// time will look like '{"@":"2020-04-12T16:34:04.123456789Z"}'
	TimeWrap string

	// TimeMap if true will encode time as a map with a create key and a
	// 'value' member formatted according to the TimeFormat options.
	TimeMap bool

	// CreateKey if set is the key to use when encoding objects that can later
	// be reconstituted with an Unmarshall call. This is only use when writing
	// simple types where one of the object in an array or map is not a
	// Simplifier. Reflection is used to encode all public members of the
	// object if possible. For example, is CreateKey is set to "type" this
	// might be the encoding.
	//
	//   { "type": "MyType", "a": 3, "b": true }
	//
	CreateKey string

	// NoReflect if true does not use reflection to encode an object. This is
	// only considered if the CreateKey is empty.
	NoReflect bool

	// FullTypePath if true includes the full type name and path when used
	// with the CreateKey.
	FullTypePath bool

	// Color if true will colorize the output.
	Color bool

	// SyntaxColor is the color for syntax in the JSON output.
	SyntaxColor string

	// KeyColor is the color for a key in the JSON output.
	KeyColor string

	// NullColor is the color for a null in the JSON output.
	NullColor string

	// BoolColor is the color for a bool in the JSON output.
	BoolColor string

	// NumberColor is the color for a number in the JSON output.
	NumberColor string

	// StringColor is the color for a string in the JSON output.
	StringColor string

	// TimeColor is the color for a time.Time in the JSON output.
	TimeColor string

	// NoColor turns the color off.
	NoColor string

	// UseTags if true will use the json annotation tags when marhsalling,
	// writing, or decomposing an struct. If no tag is present then the
	// KeyExact flag is referenced to determine the key.
	UseTags bool

	// KeyExact if true will use the exact field name for an encoded struct
	// field. If false the key style most often seen in JSON files where the
	// first character of the object keys is lowercase.
	KeyExact bool

	// HTMLUnsafe if true turns off escaping of &, <, and >.
	HTMLUnsafe bool

	// NestEmbed if true will generate an element for each anonymous embedded
	// field.
	NestEmbed bool

	// BytesAs indicates how []byte fields should be encoded. Choices are
	// BytesAsString, BytesAsBase64 (the go json package default), or
	// BytesAsArray.
	BytesAs int

	// Converter to use when decomposing or altering if non nil. The Converter
	// type includes more details.
	Converter *Converter

	// FloatFormat is the fmt.Printf formatting verb and options. The default
	// is "%g".
	FloatFormat string
}

// AppendTime appends a time string to the buffer.
func (o *Options) AppendTime(buf []byte, t time.Time, sen bool) []byte {
	if o.TimeMap {
		buf = append(buf, '{')
		if sen {
			buf = AppendSENString(buf, o.CreateKey, o.HTMLUnsafe)
		} else {
			buf = AppendJSONString(buf, o.CreateKey, o.HTMLUnsafe)
		}
		buf = append(buf, ':')
		if sen {
			if o.FullTypePath {
				buf = append(buf, `"time/Time" value:`...)
			} else {
				buf = append(buf, "Time value:"...)
			}
		} else {
			if o.FullTypePath {
				buf = append(buf, `"time/Time","value":`...)
			} else {
				buf = append(buf, `"Time","value":`...)
			}
		}
	} else if 0 < len(o.TimeWrap) {
		buf = append(buf, '{')
		if sen {
			buf = AppendSENString(buf, o.TimeWrap, o.HTMLUnsafe)
		} else {
			buf = AppendJSONString(buf, o.TimeWrap, o.HTMLUnsafe)
		}
		buf = append(buf, ':')
	}
	switch o.TimeFormat {
	case "", "nano":
		buf = strconv.AppendInt(buf, t.UnixNano(), 10)
	case "second":
		// Decimal format but float is not accurate enough so build the output
		// in two parts.
		nano := t.UnixNano()
		secs := nano / int64(time.Second)
		if 0 < nano {
			buf = append(buf, fmt.Sprintf("%d.%09d", secs, nano-(secs*int64(time.Second)))...)
		} else {
			buf = append(buf, fmt.Sprintf("%d.%09d", secs, -(nano-(secs*int64(time.Second))))...)
		}
	default:
		buf = append(buf, '"')
		buf = t.AppendFormat(buf, o.TimeFormat)
		buf = append(buf, '"')
	}
	if 0 < len(o.TimeWrap) || o.TimeMap {
		buf = append(buf, '}')
	}
	return buf
}

// DecomposeTime encodes time in the format specified by the settings of the
// options.
func (o *Options) DecomposeTime(t time.Time) (v any) {
	switch o.TimeFormat {
	case "time":
		v = t
	case "", "nano":
		v = t.UnixNano()
	case "second":
		v = float64(t.UnixNano()) / float64(time.Second)
	default:
		v = t.Format(o.TimeFormat)
	}
	if o.TimeMap {
		if o.FullTypePath {
			v = map[string]any{o.CreateKey: "time/Time", "value": v}
		} else {
			v = map[string]any{o.CreateKey: "Time", "value": v}
		}
	} else if 0 < len(o.TimeWrap) {
		v = map[string]any{o.TimeWrap: v}
	}
	return
}
