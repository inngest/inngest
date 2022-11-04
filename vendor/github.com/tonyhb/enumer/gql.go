package main

// Arguments to format are:
//	[1]: type name
const unmarshalGqlMethod = `func (i *%[1]s) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		bytes, ok := v.([]byte)
		if !ok {
			return fmt.Errorf("value is not a byte slice")
		}

		str = string(bytes[:])
	}

	val, err := %[1]sFromString(str)
	if err != nil {
		return err
	}
	
	*i = val
	return nil
}
`

const marshalGqlMethod = `func (i %[1]s) MarshalGQL(w io.Writer) {
	_, _ = w.Write([]byte(strconv.Quote(i.String())))
}
`

func (g *Generator) addGqlMethods(typeName string) {
	g.Printf("\n")
	g.Printf(unmarshalGqlMethod, typeName)
	g.Printf("\n\n")
	g.Printf(marshalGqlMethod, typeName)
}
