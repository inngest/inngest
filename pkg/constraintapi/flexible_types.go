package constraintapi

import "encoding/json"

// flexibleIntArray handles both empty objects {} and arrays []
// This is needed because Lua's cjson.empty_array returns {} instead of []
type flexibleIntArray []int

func (f *flexibleIntArray) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as array first
	var arr []int
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = flexibleIntArray(arr)
		return nil
	}
	
	// If it fails, check if it's an empty object {}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil && len(obj) == 0 {
		*f = flexibleIntArray([]int{})
		return nil
	}
	
	return json.Unmarshal(data, (*[]int)(f))
}

// flexibleStringArray handles both empty objects {} and arrays []
// This is needed because Lua's cjson.empty_array returns {} instead of []
type flexibleStringArray []string

func (f *flexibleStringArray) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = flexibleStringArray(arr)
		return nil
	}
	
	// If it fails, check if it's an empty object {}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil && len(obj) == 0 {
		*f = flexibleStringArray([]string{})
		return nil
	}
	
	return json.Unmarshal(data, (*[]string)(f))
}