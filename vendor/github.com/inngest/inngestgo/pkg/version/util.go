package version

import "fmt"

func GetVersion() string {
	return fmt.Sprintf("go:%s", SDKVersion)
}
