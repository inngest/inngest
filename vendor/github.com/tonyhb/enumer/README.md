# Enumer [![GoDoc](https://godoc.org/github.com/alvaroloes/enumer?status.svg)](https://godoc.org/github.com/alvaroloes/enumer) [![Go Report Card](https://goreportcard.com/badge/github.com/alvaroloes/enumer)](https://goreportcard.com/report/github.com/alvaroloes/enumer) [![cover.run go](https://cover.run/go/github.com/alvaroloes/enumer.svg?tag=golang-1.10)](https://cover.run/go/github.com/alvaroloes/enumer?tag=golang-1.10)
Enumer is a tool to generate Go code that adds useful methods to Go enums (constants with a specific type).
It started as a fork of [Rob Pike’s Stringer tool](https://godoc.org/golang.org/x/tools/cmd/stringer).

## Install
Enumer can be installed as any other go command:

```
go get github.com/alvaroloes/enumer
```
After that, the `enumer` executable will be in "$GOPATH/bin" folder and you can use it with `go generate`


## Generated functions and methods
When Enumer is applied to a type, it will generate:

* The following basic methods/functions: 

  * Method `String()`: returns the string representation of the enum value. This makes the enum conform
the `Stringer` interface, so whenever you print an enum value, you'll get the string name instead of a number.
  * Function `<Type>String(s string)`: returns the enum value from its string representation. This is useful
when you need to read enum values from command line arguments, from a configuration file, or
from a REST API request... In short, from those places where using the real enum value (an integer) would
be almost meaningless or hard to trace or use by a human.
  * Function `<Type>Values()`: returns a slice with all the values of the enum
  * Method `IsA<Type>()`: returns true only if the current value is among the values of the enum. Useful for validations.
* When the flag `json` is provided, two additional methods will be generated, `MarshalJSON()` and `UnmarshalJSON()`. These make
the enum conform to the `json.Marshaler` and `json.Unmarshaler` interfaces. Very useful to use it in JSON APIs.
* When the flag `text` is provided, two additional methods will be generated, `MarshalText()` and `UnmarshalText()`. These make
the enum conform to the `encoding.TextMarshaler` and `encoding.TextUnmarshaler` interfaces. 
**Note:** If you use your enum values as keys in a map and you encode the map as _JSON_, you need this flag set to true to properly
convert the map keys to json (strings). If not, the numeric values will be used instead
* When the flag `yaml` is provided, two additional methods will be generated, `MarshalYAML()` and `UnmarshalYAML()`. These make
the enum conform to the `gopkg.in/yaml.v2.Marshaler` and `gopkg.in/yaml.v2.Unmarshaler` interfaces.
* When the flag `sql` is provided, the methods for implementing the Scanner and Valuer interfaces will be also generated.
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
pill, err := PillString("Ibuprofen")
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

To transform it from CamelCase to snake_case or kebab-case, you can use the `transform` flag.

For example, the command `enumer -type=MyType -json -transform=snake` would generate the following string representation:

```go
name := MyTypeValue.String() // name => "my_type_value"
```
**Note**: The transformation only works form CamelCase to snake_case or kebab-case, not the other way around.

## How to use
The usage of Enumer is the same as Stringer, so you can refer to the [Stringer docs](https://godoc.org/golang.org/x/tools/cmd/stringer)
for more information.

There are four boolean flags: `json`, `text`, `yaml` and `sql`. You can use any combination of them (i.e. `enumer -type=Pill -json -text`),


For enum string representation transformation the `transform` and `trimprefix` flags
were added (i.e. `enumer -type=MyType -json -transform=snake`).
Possible transform values are `snake` and `kebab` for transformation to snake_case and kebab-case accordingly.
The default value for `transform` flag is `noop` which means no transformation will be performed.

If a prefix is provided via the `trimprefix` flag, it will be trimmed from the start of each name (before
it is transformed). If a name doesn't have the prefix it will be passed unchanged.

## Inspiring projects
* [Stringer](https://godoc.org/golang.org/x/tools/cmd/stringer)
* [jsonenums](https://github.com/campoy/jsonenums)
