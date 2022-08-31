package main

import (
	"encoding/json"
	"os"

	"github.com/inngest/inngestgo/actionsdk"
)

type JsonEnv struct {
}

func main() {
	var jsonExample map[string]interface{}
	json.Unmarshal([]byte(os.Getenv("JSON")), &jsonExample)

	actionsdk.WriteResult(&actionsdk.Result{
		Status: 200,
		Body: map[string]interface{}{
			"simple":        os.Getenv("SIMPLE"),
			"quoted":        os.Getenv("QUOTED"),
			"quotedEscapes": os.Getenv("QUOTED_ESCAPES"),
			"certificate":   os.Getenv("CERTIFICATE"),
			"json":          jsonExample,
		},
	})
}
