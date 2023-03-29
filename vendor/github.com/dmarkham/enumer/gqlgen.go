package main

// Arguments to format are:
//	[1]: type name
const gqlgenMethods = `
// MarshalGQL implements the graphql.Marshaler interface for %[1]s
func (i %[1]s) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(i.String()))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface for %[1]s
func (i *%[1]s) UnmarshalGQL(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("%[1]s should be a string, got %%T", value)
	}

	var err error
	*i, err = %[1]sString(str)
	return err
}
`

func (g *Generator) buildGQLGenMethods(runs [][]Value, typeName string) {
	g.Printf(gqlgenMethods, typeName)
}
