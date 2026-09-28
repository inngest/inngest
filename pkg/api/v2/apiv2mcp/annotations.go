package apiv2mcp

import "github.com/modelcontextprotocol/go-sdk/mcp"

// New methods stay conservative until their effects have been reviewed.
func toolAnnotations(method string) *mcp.ToolAnnotations {
	readOnly, destructive, idempotent, openWorld := false, true, false, true
	switch method {
	case "Health", "FetchAccount", "FetchAccountEnvs", "FetchAccountEventKeys", "FetchAccountSigningKeys",
		"ListWebhooks", "GetFunctionRun", "ListRuns", "ListFunctionRuns", "GetEventRuns",
		"GetApp", "GetApps", "ListSandboxes", "GetSandbox", "ListSandboxSnapshots", "GetSandboxSnapshot",
		"StreamSandboxLogs", "ReadSandboxFile", "ListSandboxProcesses", "GetSandboxProcess",
		"WaitSandboxProcess", "GetSandboxProcessOutput", "StreamSandboxProcessOutput",
		"GetFunctionTrace", "GetFunction", "GetFunctions", "ListInsightsTables", "ListInsightsEventSchemas",
		"QueryInsights", "ListExperiments", "GetExperiment", "ListSessionKeys", "ListSessions", "ListSessionRuns":
		readOnly, destructive, idempotent, openWorld = true, false, true, false
	case "QueryInsightsPrompt":
		readOnly, destructive, idempotent, openWorld = true, false, true, true
	case "CreateEnv", "CreateWebhook", "CreateScore", "CreateSandboxSnapshot":
		destructive, openWorld = false, false
	case "PatchEnv", "CancelRun", "DestroySandbox", "PauseSandbox", "DeleteSandboxSnapshot", "SignalSandboxProcess":
		openWorld = false
	case "WriteSandboxFile":
		openWorld = false
	case "CreateSandbox":
		destructive = false
	case "ResumeSandbox", "ExecSandbox", "StartSandboxProcess", "SyncApp", "SendEvent", "InvokeFunction", "Rerun":
	}
	return &mcp.ToolAnnotations{ReadOnlyHint: readOnly, DestructiveHint: new(destructive), IdempotentHint: idempotent, OpenWorldHint: new(openWorld)}
}
