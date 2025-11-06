package util

import "fmt"

func DataWrap(data []byte) string {
	return fmt.Sprintf(`{"data":%s}`, data)
}
