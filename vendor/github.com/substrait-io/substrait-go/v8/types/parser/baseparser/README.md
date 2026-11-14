## Type Parsing

This folder contains the parser code for the Substrait types.
The parser is generated using the ANTLR4 tool. The parser reads a Substrait type string and returns a `Type` object.
The parser is generated using the SubstraitLexer.g4 and SubstraitType.g4 files from https://github.com/substrait-io/substrait/blob/main/grammar.

### Steps to regenerate the parser code

Whenever the grammar files are updated in Substrait repo, the parser code must be regenerated. To do this follow below steps:

#### Step1:
Update the `generate.go` file in the `grammar` folder at the root of the repository to pull the new grammar files.

```
//go:generate wget https://raw.githubusercontent.com/substrait-io/substrait/<commit-hash>/grammar/SubstraitLexer.g4
//go:generate wget https://raw.githubusercontent.com/substrait-io/substrait/<commit-hash>/grammar/SubstraitType.g4
```
Replace `<commit-hash>` with the commit hash of the Substrait repo that contains the updated grammar files.

#### Step2:
To generate the parser code, run the following command in the root of the repository:

```
go generate ./grammar/...
```
