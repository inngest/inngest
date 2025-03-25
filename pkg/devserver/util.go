package devserver

import "fmt"

func anyToString(value any) (string, error) {
	var res string

	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		res = fmt.Sprintf("%d", v)
	case bool:
		res = fmt.Sprintf("%t", v)
	case string:
		res = v
	default:
		return "", fmt.Errorf("unexpected data type detected: %v", v)
	}

	return res, nil
}

func convertMap(src map[string]any) (map[string]string, error) {
	res := map[string]string{}

	for k, v := range src {
		val, err := anyToString(v)
		if err != nil {
			return nil, err
		}
		res[k] = val
	}
	return res, nil
}
