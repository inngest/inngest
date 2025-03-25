# Simple Encoding Notation

SEN (Simple Encoding Notation) is a simple minimal encoding
notation. Drawing from JSON but less strict. A SEN parse must be able
to parse JSON but should also ignore commas and should allow tokens to
be read as strings where a token is a set of characters described
by:

```ebnf
token         = tokenStart [{tokenContinue}]
tokenStart    = letter | "_" | "^" | "."
tokenContinue = tokenStart | digit | "-"
letter        = [A-Za-z_^~.] | U+0080 - U+FFFFFFFF
digit         = [0-9]
```

Charcter encoding is Unicode and the stream or file encoding must be
UTF-8. A UTF-8 BOM at the start of a sequence is allowed.

C style comments that start with a `//` sequence are allowed and
ignored.

Strings can also be delimited with a single quote character which
allows for a string to be either `"abc"` or `'abc'`.

In all other aspects SEN is as described by the
[json.org](https://www.json.org/json-en.html) description.

A valid example of a SEN document is:

```
{
  one: 1
  two: 2
  array: [a b c]
  yes: true
}
```

Which is the same as the following JSON:

```json
{
  "one": 1,
  "two": 2,
  "array": ["a", "b", "c"],
  "yes": true
}
```
