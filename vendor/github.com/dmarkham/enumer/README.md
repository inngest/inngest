# Enumer [![GoDoc](https://godoc.org/github.com/dmarkham/enumer?status.svg)](https://godoc.org/github.com/dmarkham/enumer) [![Go Report Card](https://goreportcard.com/badge/github.com/dmarkham/enumer)](https://goreportcard.com/report/github.com/dmarkham/enumer) [![GitHub Release](https://img.shields.io/github/release/dmarkham/enumer.svg)](https://github.com/dmarkham/enumer/releases)[![Build Status](https://travis-ci.com/dmarkham/enumer.svg?branch=master)](https://travis-ci.com/dmarkham/enumer)


Enumer is a tool to generate Go code that adds useful methods to Go enums (constants with a specific type).
It started as a fork of [Rob Pike’s Stringer tool](https://godoc.org/golang.org/x/tools/cmd/stringer)
maintained by [Álvaro López Espinosa](https://github.com/alvaroloes/enumer). 
This was again forked here as (https://github.com/dmarkham/enumer) picking up where Álvaro left off.


```
$ ./enumer --help
Enumer is a tool to generate Go code that adds useful methods to Go enums (constants with a specific type).
Usage of ./enumer:
        Enumer [flags] -type T [directory]
        Enumer [flags] -type T files... # Must be a single package
For more information, see:
        http://godoc.org/github.com/dmarkham/enumer
Flags:
  -addprefix string
        transform each item name by adding a prefix. Default: ""
  -comment value
        comments to include in generated code, can repeat. Default: ""
  -gqlgen
        if true, GraphQL marshaling methods for gqlgen will be generated. Default: false
  -json
        if true, json marshaling methods will be generated. Default: false
  -linecomment
        use line comment text as printed text when present
  -output string
        output file name; default srcdir/<type>_string.go
  -sql
        if true, the Scanner and Valuer interface will be implemented.
  -text
        if true, text marshaling methods will be generated. Default: false
  -transform string
        enum item name transformation method. Default: noop (default "noop")
  -trimprefix string
        transform each item name by removing a prefix. Default: ""
  -type string
        comma-separated list of type names; must be set
  -values
    	if true, alternative string values method will be generated. Default: false
  -yaml
        if true, yaml marshaling methods will be generated. Default: false
```


## Generated functions and methods

When Enumer is applied to a type, it will generate:

- The following basic methods/functions:

  - Method `String()`: returns the string representation of the enum value. This makes the enum conform
    the `Stringer` interface, so whenever you print an enum value, you'll get the string name instead of a number.
  - Function `<Type>String(s string)`: returns the enum value from its string representation. This is useful
    when you need to read enum values from command line arguments, from a configuration file, or
    from a REST API request... In short, from those places where using the real enum value (an integer) would
    be almost meaningless or hard to trace or use by a human. `s` string is Case Insensitive.
  - Function `<Type>Values()`: returns a slice with all the values of the enum
  - Function `<Type>Strings()`: returns a slice with all the Strings of the enum
  - Method `IsA<Type>()`: returns true only if the current value is among the values of the enum. Useful for validations.

- When the flag `json` is provided, two additional methods will be generated, `MarshalJSON()` and `UnmarshalJSON()`. These make
  the enum conform to the `json.Marshaler` and `json.Unmarshaler` interfaces. Very useful to use it in JSON APIs.
- When the flag `text` is provided, two additional methods will be generated, `MarshalText()` and `UnmarshalText()`. These make
  the enum conform to the `encoding.TextMarshaler` and `encoding.TextUnmarshaler` interfaces.
  **Note:** If you use your enum values as keys in a map and you encode the map as _JSON_, you need this flag set to true to properly
  convert the map keys to json (strings). If not, the numeric values will be used instead
- When the flag `yaml` is provided, two additional methods will be generated, `MarshalYAML()` and `UnmarshalYAML()`. These make
  the enum conform to the `gopkg.in/yaml.v2.Marshaler` and `gopkg.in/yaml.v2.Unmarshaler` interfaces.
- When the flag `sql` is provided, the methods for implementing the `Scanner` and `Valuer` interfaces.
  Useful when storing the enum in a database.


For example, if we have an enum type called `Pill`,

```go
type Pill int

const (
	Placebo Pill = iota
	Aspirin
	Ibuprofen
	Paracetamol
	Acetaminophen = Paracetamol
)
```

executing `enumer -type=Pill -json` will generate a new file with four basic methods and two extra for JSON:

```go
func (i Pill) String() string {
	//...
}

func PillString(s string) (Pill, error) {
	//...
}

func PillValues() []Pill {
	//...
}

func PillStrings() []string {
	//...
}

func (i Pill) IsAPill() bool {
	//...
}

func (i Pill) MarshalJSON() ([]byte, error) {
	//...
}

func (i *Pill) UnmarshalJSON(data []byte) error {
	//...
}
```

From now on, we can:

```go
// Convert any Pill value to string
var aspirinString string = Aspirin.String()
// (or use it in any place where a Stringer is accepted)
fmt.Println("I need ", Paracetamol) // Will print "I need Paracetamol"

// Convert a string with the enum name to the corresponding enum value
pill, err := PillString("Ibuprofen") // "ibuprofen" will also work.
if err != nil {
    fmt.Println("Unrecognized pill: ", err)
    return
}
// Now pill == Ibuprofen

// Get all the values of the string
allPills := PillValues()
fmt.Println(allPills) // Will print [Placebo Aspirin Ibuprofen Paracetamol]

// Check if a value belongs to the Pill enum values
var notAPill Pill = 42
if (notAPill.IsAPill()) {
	fmt.Println(notAPill, "is not a value of the Pill enum")
}

// Marshal/unmarshal to/from json strings, either directly or automatically when
// the enum is a field of a struct
pillJSON := Aspirin.MarshalJSON()
// Now pillJSON == `"Aspirin"`
```

The generated code is exactly the same as the Stringer tool plus the mentioned additions, so you can use
**Enumer** where you are already using **Stringer** without any code change.

## Transforming the string representation of the enum value

By default, Enumer uses the same name of the enum value for generating the string representation (usually CamelCase in Go).

```go
type MyType int

 ...

name := MyTypeValue.String() // name => "MyTypeValue"
```

Sometimes you need to use some other string representation format than CamelCase (i.e. in JSON).

To transform it from CamelCase to another format, you can use the `transform` flag.

For example, the command `enumer -type=MyType -json -transform=snake` would generate the following string representation:

```go
name := MyTypeValue.String() // name => "my_type_value"
```

**Note**: The transformation only works from CamelCase to snake_case or kebab-case, not the other way around.

### Transformers

- snake
- snake-upper
- kebab
- kebab-upper
- lower (lowercase)
- upper (UPPERCASE)
- title (TitleCase)
- title-lower (titleCase)
- first (Use first character of string)
- first-lower (same as first only lower case)
- first-upper (same as first only upper case)
- whitespace

## How to use

For a module-aware repo with `enumer` in the `go.mod` file, generation can be called by adding the following to a `.go` source file:

```golang
//go:generate go run github.com/dmarkham/enumer -type=YOURTYPE
```

There are four boolean flags: `json`, `text`, `yaml` and `sql`. You can use any combination of them (i.e. `enumer -type=Pill -json -text`),

For enum string representation transformation the `transform` and `trimprefix` flags
were added (i.e. `enumer -type=MyType -json -transform=snake`).
Possible transform values are listed above in the [transformers](#transformers) section.
The default value for `transform` flag is `noop` which means no transformation will be performed.

If a prefix is provided via the `trimprefix` flag, it will be trimmed from the start of each name (before
it is transformed). If a name doesn't have the prefix it will be passed unchanged.

If a prefix is provided via the `addprefix` flag, it will be added to the start of each name (after trimming and after transforming).

The boolean flag `values` will additionally create an alternative string values method `Values() []string` to fullfill the `EnumValues` interface of [ent](https://entgo.io/docs/schema-fields/#enum-fields).

## Inspiring projects

- [Álvaro López Espinosa](https://github.com/alvaroloes/enumer)
- [Stringer](https://godoc.org/golang.org/x/tools/cmd/stringer)
- [jsonenums](https://github.com/campoy/jsonenums)
