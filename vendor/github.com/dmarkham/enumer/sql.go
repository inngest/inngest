package main

// Arguments to format are:
//	[1]: type name
const valueMethod = `func (i %[1]s) Value() (driver.Value, error) {
	return i.String(), nil
}
`

const scanMethod = `func (i *%[1]s) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	case fmt.Stringer:
		str = v.String()
	default:
		return fmt.Errorf("invalid value of %[1]s: %%[1]T(%%[1]v)", value)
	}

	val, err := %[1]sString(str)
	if err != nil {
		return err
	}

	*i = val
	return nil
}
`

func (g *Generator) addValueAndScanMethod(typeName string) {
	g.Printf("\n")
	g.Printf(valueMethod, typeName)
	g.Printf("\n\n")
	g.Printf(scanMethod, typeName)
}
