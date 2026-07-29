package constraintapi

import "encoding/json"

func unmarshalFlexibleArray[T any](data []byte, target *[]T) error {
	if err := json.Unmarshal(data, target); err == nil {
		return nil
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil && len(obj) == 0 {
		*target = []T{}
		return nil
	}

	return json.Unmarshal(data, target)
}

// flexibleIntArray handles both empty objects {} and arrays [].
// Redis Lua cjson encodes empty tables as {}, even when the field is
// semantically an array.
type flexibleIntArray []int

func (f *flexibleIntArray) UnmarshalJSON(data []byte) error {
	var arr []int
	if err := unmarshalFlexibleArray(data, &arr); err != nil {
		return err
	}
	*f = flexibleIntArray(arr)
	return nil
}

// flexibleStringArray handles both empty objects {} and arrays [].
// Redis Lua cjson encodes empty tables as {}, even when the field is
// semantically an array.
type flexibleStringArray []string

func (f *flexibleStringArray) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := unmarshalFlexibleArray(data, &arr); err != nil {
		return err
	}
	*f = flexibleStringArray(arr)
	return nil
}

type flexibleConstraintUsageArray []scriptConstraintUsage

func (f *flexibleConstraintUsageArray) UnmarshalJSON(data []byte) error {
	var arr []scriptConstraintUsage
	if err := unmarshalFlexibleArray(data, &arr); err != nil {
		return err
	}
	*f = flexibleConstraintUsageArray(arr)
	return nil
}

type checkScriptConstraintUsage struct {
	Usage int `json:"u"`
	Limit int `json:"l"`
}

type flexibleCheckConstraintUsageArray []checkScriptConstraintUsage

func (f *flexibleCheckConstraintUsageArray) UnmarshalJSON(data []byte) error {
	var arr []checkScriptConstraintUsage
	if err := unmarshalFlexibleArray(data, &arr); err != nil {
		return err
	}
	*f = flexibleCheckConstraintUsageArray(arr)
	return nil
}
