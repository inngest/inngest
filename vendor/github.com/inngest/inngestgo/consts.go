package inngestgo

import (
	"runtime/debug"
)

const (
	SDKAuthor         = "inngest"
	SDKLanguage       = "go"
	SyncKindInBand    = "in_band"
	SyncKindOutOfBand = "out_of_band"
)

const (
	devServerOrigin       = "http://127.0.0.1:8288"
	defaultAPIOrigin      = "https://api.inngest.com"
	defaultEventAPIOrigin = "https://inn.gs"
)

const (
	executionVersionV2 = "2"
)

var (
	SDKVersion = ""
)

func init() {
	readBuildInfo()
}

func readBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	/*
		Find and set the SDK version.

		When imported into another project, its value will be something like
		"v0.7.5-0.20250305172920-ddde6dd6f565".

		When run within this project, it'll be "(devel)".
	*/
	const modulePath = "github.com/inngest/inngestgo"
	for _, dep := range info.Deps {
		if dep.Path == modulePath && dep.Version != "" {
			SDKVersion = dep.Version
			break
		}
	}
}
