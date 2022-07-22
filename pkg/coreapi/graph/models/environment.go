package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Environment string

const (
	Production Environment = "prod"
	Test       Environment = "test"
)

var environmentMap = map[string]Environment{
	"prod": Production,
	"test": Test,
}

func (e Environment) String() string {
	if e == Test {
		return string(Test)
	}
	return string(Production)
}

func ParseString(s string) (Environment, error) {
	if val, ok := environmentMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return "", fmt.Errorf("%s is not a valid environment", s)
}

func (e *Environment) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Environment should be a string, got %s", data)
	}
	var err error
	*e, err = ParseString(s)
	return err
}

func (e *Environment) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		bytes, ok := v.([]byte)
		if !ok {
			return fmt.Errorf("value is not a byte slice")
		}
		str = string(bytes[:])
	}
	val, err := ParseString(str)
	if err != nil {
		return err
	}
	*e = val
	return nil
}
