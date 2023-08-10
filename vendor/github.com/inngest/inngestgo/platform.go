package inngestgo

import (
	"fmt"
	"os"
)

func platform() string {
	// TODO: Better Platform detection, eg. vercel, lambda.
	if region := os.Getenv("AWS_REGION"); region != "" {
		return fmt.Sprintf("aws-%s", region)
	}
	if os.Getenv("VERCEL") != "" {
		return "vercel"
	}
	if os.Getenv("NETLIFY") != "" {
		return "netlify"
	}
	return ""
}
