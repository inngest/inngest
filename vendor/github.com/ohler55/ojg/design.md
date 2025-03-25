# A Journey building a fast JSON parser and full JSONPath, Oj for Go

I had a dream. I'd write a fast JSON parser, generic data, and a
JSONPath implementation and it would be beautiful, well organized, and
something to be admired. Well, reality kicked in and laughed at those
dreams. A Go JSON parser and tools could be high performance but to
get that performance compromises in beauty would have to be made. This
is a tale of journey that ended with a Parser that leaves the Go JSON
parser in the dust and resulted in some useful tools including a
complete and efficient JSONPath implementation.

In all fairness I did embark on with some previous experience. Having
written two JSON parser before. Both the Ruby
[Oj](https://github.com/ohler55/oj) and the C parser
[OjC](https://github.com/ohler55/ojc). Why not an
[OjG](https://github.com/ohler55/ojg) for go.

## Planning

Like any journey it starts with the planning. Yeah, I know, it's called
requirement gathering but casting it as planning a journey is more fun
and this was all about enjoying the discoveries on the journey. The
journey takes place in the land of OjG which stands for Oj for
Go. [Oj](https://github.com/ohler55/oj) or Optimized JSON being a
popular gem I wrote for Ruby.

First, JSON parsing and any frequently used operations such as
JSONPath evaluation had to be fast over everything else. With the
luxury of not having to follow the existing Go json package API the
API could be designed for the best performance.

The journey would visit several areas each with its own landscape and
different problems to solve.

### Generic Data

The first visit was to generic data. Not to be confused with the
proposed Go generics. Thats a completely different animal and has
nothing to do with whats being referred to as generic data here. In
building tools or packages for reuse the data acted on by those tools
needs to be navigable.

Reflection can be used but that gets a bit tricky when dealing with
private fields or field that can't be converted to something that can
say be written as a JSON element. Other options are often better.

Another approach is to use simple Go types such as `bool`, `int64`,
`[]any`, and other types that map directly on to JSON or some
other subset of all possible Go types. If too open, such as with
`[]any` it is still possible for the user to put unsupported
types into the data. Not to pick out any package specifically but it
is frustrating to see an argument type of `any` in an API and
then no documentation describing that the supported types are.

There is another approach though: Define a set of types that can be in
a collection and use those types. With this approach, the generic data
implementation has to support the basic JSON types of `null`,
`boolean`, `int64`, `float64`, `string`, array, and object. In
addition time should be supported. From experience in both JSON use in
Ruby and Go time has always been needed. Time is just too much a part
of any set of data to leave it out.

The generic data had to be type safe. It would not do to have an
element that could not be encoded as JSON in the data.

A frequent operation for generic data is to store that data into a
JSON database or similar. That meant converting to simple Go types of
`nil`, `bool`, `int64`, `float64`, `string`, `[]any`, and
`map[string]any` had to be fast.

Also planned for this part of the journey was methods on the types to
support getting, setting, and deleting elements using JSONPath. The
hope was to have an object based approach to the generic nodes so
something like the following could be used but keeping generic data,
JSONPath, and parsing in separate packages.

```golang
    var n gen.Node
    n = gen.Int(123)
    i, ok := n.AsInt()
```

Unfortunately that part of the journey had to be cancelled as the Go
travel guide refuses to let packages talk back and forth. Imports are
one way only. After trying to put all the code in one package it
eventually got unwieldy. Function names started being prefixed with
what should really have been package names so the object and method
approach was dropped. A change in API but the journey would continue.

### JSON Parser and Validator

The next stop was the parser and validator. After some consideration
it seemed like starting with the validator would be best way to become
familiar with the territory. The JSON parser and validator need not be
the same and each should be as performant as possible. The parsers
needed to support parsing into simple Go types as well as the generic
data types.

When parsing files that include millions or more JSON elements in
files that might be over 100GB a streaming parser is necessary. It
would be nice to share some code with both the streaming and string
parsers of course. It's easier to pack light when the areas are
similar.

The parser must also allow parsing into native Go types. Furthermore
interfaces must be supported even though Go unmarshalling does not
support interface fields. Many data types make use of interfaces
that limitation was not acceptable for the OjG parser. A different
approach to support interfaces was possible.

JSON documents of any non-trivial size, especially if hand-edited, are
likely to have errors at some point. Parse errors must identify where
in the document the error occurred.

### JSONPath

Saving the most interesting part of the trip for last, the JSONPath
implementation promised to have all sorts of interesting problems to
solve with descents, wildcards, and especially filters.

A JSONPath is used to extract elements from data. That part of the
implementation had to be fast. Parsing really didn't have to be fast
but it would be nice to have a way of building a JSONPath in a
performant manner even if it was not as convenient as parsing a
string.

The JSONPath implementation had to implement all the features
described by the [Goessner
article](https://goessner.net/articles/JsonPath). There are other
descriptions of JSONPath but the Goessner description is the most
referenced. Since the implementation is in Go the scripting feature
described could be left out as long as similar functionality could be
provided for array indexes relative to the length of the
array. Borrowing from Ruby, using negative indexes would provide that
functionality.

## The Journey

The journey unfolded as planned to a degree. There were some false
starts and revisits but eventually each destination was reached and
the journey completed.

### Generic Data (`gen` package)

What better way to make generic type fast than to just define generic
types from simple Go types and then add methods on those types? A
`gen.Int` is just an `int64` and a `gen.Array` is just a
`[]gen.Node`. With that approach there are no extra allocations.

```golang
type Node any
type Int int64
type Array []Node
```

Since generic arrays and objects restrict the type of the values in
each collection to `gen.Node` types the collections are assured to
contain only elements that can be encoded as JSON.

Methods on the `Node` could not be implemented without import loops so
the number of functions in the `Node` interface were limited. It was
clear a parser specific to the generic data type would be needed but
that would have to wait until the parser part of the journey was
completed. Then the generic data package could be revisited and the
parser explored.

Peeking at the future to the generic data parser revisit it was not
very interesting after the deep dive into the simple data parser. The
parser for generic types is a copy of the oj package parser but
instead of simple types being created instances that support the
`gen.Node` interface are created.

### Simple Parser (`oj` package)

Looking back its hard to say what was the most interesting part of the
journey, the parser or JSONPath. Each had their own unique set of
issues. The parser was the best place to start though as some valuable
lessons were learned about what to avoid and what to gravitate toward
in trying to achieve high performance Go code.

#### Validator

From the start I knew that a single pass parser would be more
efficient than building tokens and then making a second pass to decide
what the tokens means. At least that approach as worked well in the
past. I dived in and used a `readValue` function that branched
depending on the next character read. It worked but it was slower than
the target of being on par with the Go `json.Validate`. That was the
bar to clear. The first attempt was off by a lot. Of course a
benchmark was needed to verify that so the `cmd/benchmark` command was
started. Profiling didn't help much. It turned out since much of the
overhead was in the function call setup which isn't obvious when
profiling.

Not knowing at the time that function calls were so expensive but
anticipating that there was some overhead in function calls I moved
some of the code from a few frequently called functions to be inline
in the calling function. That made much more of a difference than I
expected. At that point I looked at the Go code for the core
validation code. I was surprised to see that it used lots of functions
but not functions attached to a type. I gave that approach a try but
with functions on the parser type. The results were not good
either. Simply changing the functions to take the parser as an
argument made a big difference though. Another lesson learned.

Next was to remove function calls as much as possible since they did
seem to be expensive. The code was no longer elegant and had lots of
duplicated blocks but it ran much faster. At this point the code
performance was getting closer to clearing the Go validator bar.

When parsing in a single pass a conceptual state machine is generally
used. When branching with functions there is still a state machine but
the states are limited for each function making it much easier to
follow. Moving into a single function meant tracking many more states
in single state machine. Implementation was with lengthy switch
statements. One problem remained though. Array and Object had to be
tracked to know when a closing `]` or `}` was allowed. Since function
calls were being avoided that meant maintaining a stack in the single
parser function. That approach worked well with very little overhead.

Another tweak was to reuse memory. At this point the parser only
allocated a few objects but why would it need to allocate any if the
buffers for the stack could be reused. That prompted a change in the
API. The initial API was for a single `Validate()` function. If the
validator was made public it could be reused. That made a lot of sense
since often similar data is parsed or validated by the same
application. That change was enough to reduce the allocations per
validation to zero and brought the performance under the Go
`json.Valid()` bar.

#### Parser

With the many optimum techniques identified while visiting the
validator, the next part of the journey was to use those same
technique on the parser.

The difference between the validator and the parser is that the parser
needs to build up data elements. The first attempt was to add the
bytes associated with a value to a reusable buffer and then parse that
buffer at the end of the value bytes in the source. It worked and was
as fast as the `json.Unmarshall` function but that was not enough as
there were still more allocations than seemed necessary.

By expanding the state machine `null`, `true`, and `false` could be
identified as values without adding to the buffer. That gave a
bit of improvement.

Numbers, specifically integers, were another value type that really
didn't need to be parsed from a buffer so instead of appending bytes
to a buffer and calling `strconv.ParseInt()`, integer values were
built as an `int64` and grown as bytes were read. If a `.` character
is encountered then the number is a decimal so the type expected is
changed and each part of a float is captured as integers and finally a
float64 is created when done. This was another improvement in
performance.

Not much could be done to improve string parsing since it is really
just appending bytes to a buffer and making them a string at the final
`"`. Each byte being appended needed to be checked though. A byte map
in the form of a 256 long bytes array is used for that purpose.

Going back to the stack used in the validator, instead of putting a
simple marker on the stack like the validator, when an Object start
character, a `{` is encountered a new `map[string]any` is put
on the stack. Values and keys are then used to set members of the
map. Nothing special there.

Saving the best for last, arrays were tougher to deal with. A value is
not just added to an array but rather appended to an array and a
potentially new array is returned. Thats not a terribly efficient way to
build a slice as it will go through multiple reallocations. Instead, a
second slice index stack is kept. As an array is to be created, a spot
is reserved on the stack and the index of that stack location is
placed on the slice index stack. After that values are pushed onto the
stack until an array close character `]` is reached. The slice index
is then referenced and a new `[]any` is allocated for all the
values from the arry index on the stack to the end of the
stack. Values are copied to the new array and the stack is collapsed
to the index. A bit complicated but it does save multiple object
allocations.

After some experimentation it turned out that the overhead of some
operations such as creating a slice or adding a number were not
impacted to any large extent by making a function call since it does
not happen as frequently as processing each byte. Some use of
functions could therefor be used to remove duplicate code without
incurring a significant performance impact.

One stop left at the parser package tour. Streaming had to be
supported. At this point there were already plans on how to deal with
streaming which was to load up a buffer and iterate over that buffer
using the exact same code as for parsing bytes and repeat until there
was nothing left to read. It seemed like using an index into the
buffer would be easier to keep track of but switching from a `for`
`range` to `for i = 0; i < size; i++ {` dropped the performance
considerably. Clearly staying with the `range` approach was
better. Once that was working a quick trip back to the validator to
allow it to support streams was made.

Stream parsing or parsing a string with multiple JSON documents in it
is best handled with a callback function. That allows the caller to
process the parsed document and move on without incurring any
additional memory allocations unless needed.

The stay at the validator and parser was fairly lengthy at a bit over
a month of evening coding.

### JSONPath (`jp` package)

The visit to JSONPath would prove to be a long stay as well with a lot
more creativity for some tantalizing problems.

The first step was to get a language and cultural refresher on
JSONPath terms and behavior. From that it was decided that a JSONPath
would be represented by a `jp.Expr` which is composed of fragments or
`jp.Frag` objects. Keeping with the guideline of minimizing
allocations the `jp.Expr` is just a slice of `jp.Frag`. In most cases
expressions are defined statically so the parser need not be fast. No
special care was taken to make the JSONPath parser fast. Instead
functions are used in an approach that is easier to understand. I said
easier, not easy. There are a fair number of dangerous curves with
trying to support bracketed notation as well as dot notation and how
that all plays nicely with the script parser so that one can call the
other to support nested filters. It was rewarding to see it all come
together though.

If the need exists to create expressions at run time then functions
are used that allow them to be constructed more easily. That makes for
a lot of functions. I also like to be able to keep code compact and
figured others might too so each fragment type can also be created
with a single letter function. They don't have to be used but they
exist to support building expressions as a chain.

```golang
    x := jp.R().D().C("abc").W().C("xyz").N(3)
    fmt.Println(x.String())
    // $..abc.*.xyz[3]
```

contrasted with the use of the JSONPath parser:

```golang
    x, err := jp.ParseString("$..abc.*.xyz[3]")
    // check err first
    fmt.Println(x.String())
    // $..abc.*.xyz[3]
```

Evaluating an expression against data involves walking down the data
tree to find one or more elements. Conceptually each fragment of a
path sets up zero or more paths to follow through the data. When the
last fragment is reached the search is done. A recursive approach
would be ideal where the evaluation of one fragment then invokes the
next fragment's eval function with as many paths it matches. Great on
paper but for something like a descent fragment (`..`) that is a lot
of function calls.

Given that function calls are expensive and slices are cheap a Forth
(the language) evaluation stack approach is used. Not exactly Forth
but a similar concept mixing data and operators. Each fragment takes
its matches and those matches already on the stack. Then the next
fragment evaluates each in turn. This continues until the stack
shrinks back to one element indicating the evaluation is complete. The
last fragment puts any matches on a results list which is returned
instead of on the stack.

 | Stack  | Frag  |
 | ------ | ----- |
 | {a:3}  | data  |
 | 'a'    | Child |

One fragment type is a filter which looks like `[?(@.x == 3)]`. This
requires a script or function evaluation. A similar stack based
approach is used for evaluating scripts. Note that scripts can and
almost always contain a JSONPath expression starting with a `@`
character. An interesting aspect of this is that a filter can contain
other filters. OjG supports nested filters.

The most memorable part of the JSONPath part of the journey had to be
the evaluation stack. That worked out great and was able to support
all the various fragment types.

### Converting or Altering Data (`alt` package)

A little extra was added to the journey once it was clear the generic
data types would not support JSONPath directly. The original plan was
to has functions like `AsInt()` as part of the `Node` interface. With
that no longer reasonable an `alt` package became part of the
journey. It would be used for converting types as well as altering
existing ones. To make the last part of the trip even more interesting
the `alt` package is where marshalling and unmarshalling types came
into play but under the names of recompose and decompose since
operations were to take Go types and decompose those objects into
simple or generic data. The reverse is to recompose the simple data
back into their original types. This takes an approach used in Oj for
Ruby when the type name is encoded in the decomposed data. Since the
data type is included in the data itself it is self describing and can
be used to recompose types that include interface members.

There is a trade off in that JSON is not parsed directly to a Go type
by must go through an intermediate data structure first. There is an
up side to that as well though. Now any simple or generic data can be
used to recompose objects and not just JSON strings.

The `alt.GenAlter()` function was interesting in that it is possible
to modify a slice type and then reset the members without
reallocating.

Thats the last stop of the journey.

## Lessons Learned

Benchmarking was instrumental to tuning and picking the most favorable
approach to the implementation. Through those benchmarks a number of
lessons were learned.  The final benchmarks results can be viewed by
running the `cmd/benchmark` command. See the results at
[benchmarks.md](benchmarks.md).

Here is a snippet from the benchmarks. Note higher is better for the
numbers in parenthesis which is a ratio of the OjG component to Go
json package component.

```
Parse JSON
json.Unmarshal:           7104 ns/op (1.00x)    4808 B/op (1.00x)      90 allocs/op (1.00x)
  oj.Parse:               4518 ns/op (1.57x)    3984 B/op (1.21x)      86 allocs/op (1.05x)
  oj.GenParse:            4623 ns/op (1.54x)    3984 B/op (1.21x)      86 allocs/op (1.05x)

json.Marshal:             2616 ns/op (1.00x)     992 B/op (1.00x)      22 allocs/op (1.00x)
  oj.JSON:                 436 ns/op (6.00x)     131 B/op (7.57x)       4 allocs/op (5.50x)
  oj.Write:                455 ns/op (5.75x)     131 B/op (7.57x)       4 allocs/op (5.50x)
```

### Functions Add Overhead

Sure we all know a function call add some overhead in any language. In
C that overhead is pretty small or nonexistent with inline
functions. That is not true for Go. There is considerable overhead in
making a function call and if that functional call included any kind
of context such as being the function of a type the overhead is even
higher. That observation (while disappointing) drove a lot of the
parser and JSONPath evaluation code. For nice looking and well
organized code using functions are highly recommended but for high
perfomance find a way to reduce function calls.

The implementation of the parser included a lot of duplicate code to
reduce function calls and it did make a significant difference in
performance.

The JSONPath evaluation takes an additional approach. It includes a
fair amount of code duplication but it also implements its own stack
to avoid nested functional calls even though the nature of the
evaluation is a better match for a recursive implementation.

### Slices are Nice

Slices are implemented very efficiently in Go. Appending to a slice
has very little overhead. Reusing slices by collapsing them to zero
length is a great way to avoid allocating additional memory. Care has
to be taken when collapsing though as any cells in the slice that
point to objects will now leave those objects dangling or rather
referenced but not reachable and they will never be garbage
collected. Simply setting the slice slot to `nil` will avoid memory
leaks.

### Memory Allocation

Like most languages, memory allocation adds overhead. It's best to avoid
when possible. A good example of that is in the `alt` package. The
`Alter()` function replaces slice and map members instead of
allocating a new slice or map when possible.

Parsers take advantage by reusing buffers and avoiding allocation of
token during possible when possible.

### Range Has Been Optimized

Using a `for` `range` loop is better than incrementing an index. The
difference was not huge but was noticable.

### APIs Matter

It's important to define an API that is easy to use as well as one
that allows for the best performance. The parser as well as the
JSONPath builders attempt to do both. An even better example is the
[GGql](https://github.com/uhn/ggql) GraphQL package. It provides a
very simple API when compared to previous Go GraphQL packages and it
is many times
[faster](https://github.com/the-benchmarker/graphql-benchmarks).

## Whats Next?

Theres alway something new ready to be explored. For OjG there are a few things in the planning stage.

 - A short trip to Regex filters for JSONPath.
 - A construction project to add JSON building to the **oj** command which is an alternative to jq but using JSONPath.
 - Explore new territory by implementing a Simple Encoding Notation which mixes GraphQL syntax with JSON for simpler more forgiving format.
 - A callback parser along the lines of the Go json.Decoder or more likely like the Oj [Simple Callback Parser](http://ohler.com/oj/doc/Oj.html#sc_parse-class_method).

Discuss this on [Changelog News](https://changelog.com/news/a-journey-building-a-fast-json-parser-and-full-jsonpath-oj-for-go-YRXJ).
