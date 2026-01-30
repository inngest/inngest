package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/internal/embeddocs"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/oklog/ulid/v2"
)

// getDocsFS returns the embedded docs filesystem or falls back to local filesystem
func getDocsFS() fs.FS {
	// Try embedded first
	if docsFS, err := embeddocs.GetDocsFS(); err == nil {
		return docsFS
	}
	// Fall back to local filesystem if embed fails (development mode)
	return os.DirFS("docs")
}

// readEmbeddedFile reads a file from the embedded docs filesystem
func readEmbeddedFile(path string) ([]byte, error) {
	docsFS := getDocsFS()
	return fs.ReadFile(docsFS, path)
}

// walkEmbeddedDocs walks the embedded docs filesystem
func walkEmbeddedDocs(fn func(path string, d fs.DirEntry, err error) error) error {
	docsFS := getDocsFS()
	return fs.WalkDir(docsFS, ".", fn)
}

// getCachedFileContent returns cached file content or reads and caches it
func (h *MCPHandler) getCachedFileContent(path string) ([]byte, error) {
	h.fileCacheMu.RLock()
	if content, exists := h.fileCache[path]; exists {
		h.fileCacheMu.RUnlock()
		return content, nil
	}
	h.fileCacheMu.RUnlock()

	// Read file content
	content, err := readEmbeddedFile(path)
	if err != nil {
		return nil, err
	}

	// Cache the content
	h.fileCacheMu.Lock()
	h.fileCache[path] = content
	h.fileCacheMu.Unlock()

	return content, nil
}

// getCachedFileInfo returns cached file info or reads and caches it
func (h *MCPHandler) getCachedFileInfo(path string) (fs.FileInfo, error) {
	h.fileCacheMu.RLock()
	if info, exists := h.fileInfoCache[path]; exists {
		h.fileCacheMu.RUnlock()
		return info, nil
	}
	h.fileCacheMu.RUnlock()

	// Get file info
	docsFS := getDocsFS()
	info, err := fs.Stat(docsFS, path)
	if err != nil {
		return nil, err
	}

	// Cache the info
	h.fileCacheMu.Lock()
	h.fileInfoCache[path] = info
	h.fileCacheMu.Unlock()

	return info, nil
}

const (
	// Default polling configuration
	defaultPollTimeoutSeconds = 30
	defaultPollIntervalMs     = 1000

	// MCP implementation details
	mcpName    = "inngest-dev"
	mcpVersion = "v1.0.0"
	mcpTitle   = "Inngest Dev Server MCP Tools"

	// Run status constants
	statusCompleted  = "Completed"
	statusSkipped    = "Skipped"
	statusFailed     = "Failed"
	statusCancelled  = "Cancelled"
	statusOverflowed = "Overflowed"
)

type MCPHandler struct {
	events api.EventHandler
	data   cqrs.Manager
	tick   time.Duration

	serverOnce sync.Once
	server     *mcp.Server

	// File content cache
	fileCacheMu sync.RWMutex
	fileCache   map[string][]byte

	// File info cache
	fileInfoCache map[string]fs.FileInfo
}

// isRunCompleted checks if a run status indicates completion (success or skipped)
func isRunCompleted(status string) bool {
	return status == statusCompleted || status == statusSkipped
}

// isRunFailed checks if a run status indicates failure
func isRunFailed(status string) bool {
	return status == statusFailed || status == statusCancelled || status == statusOverflowed
}

// NewMCPHandler creates a new MCP handler for the dev server
func NewMCPHandler(events api.EventHandler, data cqrs.Manager, tick time.Duration) http.Handler {
	h := &MCPHandler{
		events:        events,
		data:          data,
		tick:          tick,
		fileCache:     make(map[string][]byte),
		fileInfoCache: make(map[string]fs.FileInfo),
	}

	// Create a streamable HTTP handler that returns the same server for all requests
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return h.getMCPServer()
	}, &mcp.StreamableHTTPOptions{
		JSONResponse: true, // Use JSON responses for better compatibility
	})
}

// getMCPServer returns the cached MCP server, creating it once if needed
func (h *MCPHandler) getMCPServer() *mcp.Server {
	h.serverOnce.Do(func() {
		h.server = h.createMCPServer()
	})
	return h.server
}

// createMCPServer creates an MCP server instance with dev server tools
func (h *MCPHandler) createMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    mcpName,
		Version: mcpVersion,
		Title:   mcpTitle,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "send_event",
		Description: "Send an event to the Inngest dev server which will trigger any functions listening to that event. Returns event ID and run IDs of triggered functions. Parameters: name (required string - the event name like 'test/hello.world'), data (optional - the event data, must be a JSON object or will be wrapped in {\"value\": data}), user (optional JSON object - user context), eventIdSeed (optional string for deterministic event IDs)",
	}, h.sendEvent)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_functions",
		Description: "List all registered functions in the dev server",
	}, h.listFunctions)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_run_status",
		Description: "Get detailed status and trace information for a specific function run. Parameters: runId (required string - the run ID returned from send_event or found in logs)",
	}, h.getRunStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "poll_run_status",
		Description: "Poll multiple function runs until they complete or timeout. Returns detailed status for all runs. Parameters: runIds (required array of strings - run IDs to poll), timeout (optional int - seconds to poll, default 30), pollInterval (optional int - milliseconds between polls, default 1000)",
	}, h.pollRunStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "invoke_function",
		Description: "Directly invoke a specific function and wait for its result. Unlike send_event (which is fire-and-forget), this waits for completion and returns the function's actual output data. Parameters: functionId (required string - function slug, ID, or name), data (optional - function input data, must be a JSON object or will be wrapped in {\"value\": data}), user (optional JSON object - user context), timeout (optional int - seconds to wait, default 30)",
	}, h.invokeFunction)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "grep_docs",
		Description: "Search documentation using exact string matching (grep). Useful for finding specific API names, error codes, or identifiers. Parameters: pattern (required string - the search pattern, regex supported), limit (optional int - maximum results, default 10)",
	}, h.grepDocs)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_doc",
		Description: "Read the full content of a specific documentation file. Parameters: path (required string - the doc file path relative to docs directory)",
	}, h.readDoc)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_docs",
		Description: "List all available documentation categories and their document counts. No parameters required.",
	}, h.listDocs)

	return server
}

// SendEventArgs represents the arguments for sending an event
type SendEventArgs struct {
	Name        string         `json:"name"`
	Data        map[string]any `json:"data,omitempty"`
	User        map[string]any `json:"user,omitempty"`
	EventIDSeed string         `json:"eventIdSeed,omitempty"`
}

// InvokeFunctionArgs represents the arguments for invoking a function
type InvokeFunctionArgs struct {
	FunctionID string         `json:"functionId"`
	Data       map[string]any `json:"data,omitempty"`
	User       map[string]any `json:"user,omitempty"`
	Timeout    int            `json:"timeout,omitempty"`
}

// SendEventResult represents the result of sending an event
type SendEventResult struct {
	EventID string   `json:"eventId"`
	RunIDs  []string `json:"runIds,omitempty"`
	Status  string   `json:"status"`
	Message string   `json:"message"`
}

// InvokeFunctionResult represents the result of invoking a function
type InvokeFunctionResult struct {
	RunID        string         `json:"runId"`
	FunctionName string         `json:"functionName"`
	Status       string         `json:"status"`
	Output       map[string]any `json:"output,omitempty"`
	Duration     int64          `json:"duration"`
	EventID      string         `json:"eventId"`
	Error        *string        `json:"error,omitempty"`
}

// GrepDocsArgs represents the arguments for grep_docs tool
type GrepDocsArgs struct {
	Pattern string `json:"pattern"`
	Limit   int    `json:"limit,omitempty"`
}

// GrepDocsResult represents the result of grep_docs tool
type GrepDocsResult struct {
	Matches []GrepMatch `json:"matches"`
	Count   int         `json:"count"`
}

// GrepMatch represents a single grep match
type GrepMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// ReadDocArgs represents the arguments for read_doc tool
type ReadDocArgs struct {
	Path string `json:"path"`
}

// ReadDocResult represents the result of read_doc tool
type ReadDocResult struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Content string `json:"content"`
}

// ListDocsResult represents the result of list_docs tool
type ListDocsResult struct {
	GeneratedAt string         `json:"generatedAt"`
	TotalDocs   int            `json:"totalDocs"`
	TotalChunks int            `json:"totalChunks"`
	Categories  map[string]int `json:"categories"`
	SDKs        map[string]int `json:"sdks"`
}

func (h *MCPHandler) sendEvent(ctx context.Context, req *mcp.CallToolRequest, args SendEventArgs) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "send_event"
	metadata.Context["event_name"] = args.Name
	if args.EventIDSeed != "" {
		metadata.Context["has_event_id_seed"] = true
	}
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	evt := event.Event{
		Name: args.Name,
		Data: args.Data,
		User: args.User,
	}

	// Create seed for event ID
	var seed *event.SeededID
	if args.EventIDSeed != "" {
		seed = event.SeededIDFromString(args.EventIDSeed, 0)
	}

	// Send the event via the event handler
	evtID, err := h.events(ctx, &evt, seed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send event: %w", err)
	}

	// Parse the event ID to get run IDs
	eventULID, err := ulid.Parse(evtID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse event ID: %w", err)
	}

	// Wait a moment for the event to be processed and runs to be created
	time.Sleep(h.tick * 3)

	// Get the function runs triggered by this event using existing CQRS interface
	var runIDs []string
	runs, err := h.data.GetEventRuns(ctx, eventULID, consts.DevServerAccountID, consts.DevServerEnvID)
	if err == nil && runs != nil {
		for _, run := range runs {
			runIDs = append(runIDs, run.RunID.String())
		}
	}

	// Return success result
	result := &SendEventResult{
		EventID: evtID,
		RunIDs:  runIDs,
		Status:  "accepted",
		Message: fmt.Sprintf("Event '%s' sent successfully", args.Name),
	}

	responseText := result.Message + "\nEvent ID: " + result.EventID
	if len(result.RunIDs) > 0 {
		responseText += fmt.Sprintf("\nTriggered %d function run(s): %v", len(result.RunIDs), result.RunIDs)
	} else {
		responseText += "\nNo functions were triggered by this event"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: responseText,
			},
		},
	}, result, nil
}

// ListFunctionsResult represents the result of listing functions
type ListFunctionsResult struct {
	Functions []FunctionInfo `json:"functions"`
	Count     int            `json:"count"`
}

// FunctionInfo represents basic function information
type FunctionInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Slug    string   `json:"slug"`
	Trigger []string `json:"triggers"`
}

// listFunctions handles the list_functions tool
func (h *MCPHandler) listFunctions(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "list_functions"
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	// Get all functions using existing CQRS interface
	functions, err := h.data.GetFunctions(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list functions: %w", err)
	}

	// Convert to our result format
	funcList := make([]FunctionInfo, len(functions))
	for i, fn := range functions {
		triggers := make([]string, 0)

		// Parse the function config to get trigger information
		inngestFn, err := fn.InngestFunction()
		if err == nil && inngestFn != nil {
			// Add event and cron triggers
			for _, trigger := range inngestFn.Triggers {
				if trigger.EventTrigger != nil && trigger.EventTrigger.Event != "" {
					triggers = append(triggers, trigger.EventTrigger.Event)
				}
				if trigger.CronTrigger != nil && trigger.CronTrigger.Cron != "" {
					triggers = append(triggers, fmt.Sprintf("cron: %s", trigger.CronTrigger.Cron))
				}
			}
		}

		funcList[i] = FunctionInfo{
			ID:      fn.ID.String(),
			Name:    fn.Name,
			Slug:    fn.Slug,
			Trigger: triggers,
		}
	}

	result := &ListFunctionsResult{
		Functions: funcList,
		Count:     len(funcList),
	}

	// Create text summary
	text := fmt.Sprintf("Found %d registered functions:\n", len(funcList))
	for _, fn := range funcList {
		triggersText := ""
		if len(fn.Trigger) > 0 {
			triggersText = fmt.Sprintf(" (triggers: %s)", fmt.Sprintf("%v", fn.Trigger))
		}
		text += fmt.Sprintf("- %s%s\n", fn.Name, triggersText)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, result, nil
}

// GetRunStatusArgs represents the arguments for getting run status
type GetRunStatusArgs struct {
	RunID string `json:"runId"`
}

// PollRunStatusArgs represents the arguments for polling run status
type PollRunStatusArgs struct {
	RunIDs       []string `json:"runIds"`
	Timeout      int      `json:"timeout,omitempty"`      // Total seconds to poll (default: 30)
	PollInterval int      `json:"pollInterval,omitempty"` // Milliseconds between polls (default: 1000)
}

// PollRunStatusResult represents the result of polling run status
type PollRunStatusResult struct {
	Runs      []RunStatusResult `json:"runs"`
	Completed int               `json:"completed"`
	Running   int               `json:"running"`
	Failed    int               `json:"failed"`
}

// RunStatusResult represents detailed information about a function run
type RunStatusResult struct {
	RunID        string         `json:"runId"`
	FunctionName string         `json:"functionName"`
	Status       string         `json:"status"`
	StartedAt    string         `json:"startedAt"`
	EndedAt      *string        `json:"endedAt,omitempty"`
	EventID      string         `json:"eventId"`
	EventName    string         `json:"eventName,omitempty"`
	Steps        []StepInfo     `json:"steps"`
	Output       map[string]any `json:"output,omitempty"`
	Error        *string        `json:"error,omitempty"`
}

// StepInfo represents information about a single step in a run
type StepInfo struct {
	StepName  string  `json:"stepName"`
	StepID    string  `json:"stepId"`
	Status    string  `json:"status"`
	StartedAt string  `json:"startedAt"`
	Attempt   int     `json:"attempt"`
	Error     *string `json:"error,omitempty"`
}

// getRunStatusInternal is the internal helper that does the actual run status lookup using existing CQRS interfaces
func (h *MCPHandler) getRunStatusInternal(ctx context.Context, runIDStr string) (*RunStatusResult, error) {
	// Parse the run ID
	runID, err := ulid.Parse(runIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid run ID format: %w", err)
	}

	// Get the function run using existing CQRS interface
	run, err := h.data.GetRun(ctx, runID, consts.DevServerAccountID, consts.DevServerEnvID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	// Get the function to get its name
	fn, err := h.data.GetFunctionByInternalUUID(ctx, run.FunctionID)
	functionName := "Unknown Function"
	if err == nil && fn != nil {
		functionName = fn.Name
	}

	// Get the event information
	var eventName string
	event, err := h.data.GetEvent(ctx, run.EventID, consts.DevServerAccountID, consts.DevServerEnvID)
	if err == nil && event != nil {
		eventName = event.EventName
	}

	// Get execution history using existing CQRS interface
	history, err := h.data.GetFunctionRunHistory(ctx, runID)
	var steps []StepInfo
	if err == nil && history != nil {
		for _, hist := range history {
			if hist.StepName != nil && *hist.StepName != "" {
				stepStatus := "completed"
				var stepError *string

				// Check if there's an error in the result
				if hist.Result != nil && hist.Result.ErrorCode != nil {
					stepStatus = "failed"
					stepError = hist.Result.ErrorCode
				}

				stepID := ""
				if hist.StepID != nil {
					stepID = *hist.StepID
				}
				steps = append(steps, StepInfo{
					StepName:  *hist.StepName,
					StepID:    stepID,
					Status:    stepStatus,
					StartedAt: hist.CreatedAt.Format(time.RFC3339),
					Attempt:   int(hist.Attempt),
					Error:     stepError,
				})
			}
		}
	}

	result := &RunStatusResult{
		RunID:        runIDStr,
		FunctionName: functionName,
		Status:       run.Status.String(),
		StartedAt:    run.RunStartedAt.Format(time.RFC3339),
		EventID:      run.EventID.String(),
		EventName:    eventName,
		Steps:        steps,
	}

	if run.EndedAt != nil {
		endedStr := run.EndedAt.Format(time.RFC3339)
		result.EndedAt = &endedStr
	}

	// Parse output if available
	if run.Output != nil {
		var outputData map[string]any
		if err := json.Unmarshal(run.Output, &outputData); err == nil {
			result.Output = outputData

			// Check for error in output
			if errMsg, ok := outputData["error"].(string); ok && errMsg != "" {
				result.Error = &errMsg
			}
		}
	}

	return result, nil
}

// getRunStatus handles the get_run_status tool
func (h *MCPHandler) getRunStatus(ctx context.Context, req *mcp.CallToolRequest, args GetRunStatusArgs) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "get_run_status"
	metadata.Context["run_id"] = args.RunID
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	result, err := h.getRunStatusInternal(ctx, args.RunID)
	if err != nil {
		return nil, nil, err
	}

	// Create summary text
	text := fmt.Sprintf("Run Status: %s\n", result.Status)
	text += fmt.Sprintf("Function: %s\n", result.FunctionName)
	text += fmt.Sprintf("Started: %s\n", result.StartedAt)
	if result.EndedAt != nil {
		text += fmt.Sprintf("Ended: %s\n", *result.EndedAt)
	}
	if result.EventName != "" {
		text += fmt.Sprintf("Triggered by: %s\n", result.EventName)
	}

	if len(result.Steps) > 0 {
		text += fmt.Sprintf("\nSteps (%d):\n", len(result.Steps))
		for _, step := range result.Steps {
			text += fmt.Sprintf("  - %s: %s (attempt %d)\n", step.StepName, step.Status, step.Attempt)
			if step.Error != nil {
				text += fmt.Sprintf("    Error: %s\n", *step.Error)
			}
		}
	}

	if result.Error != nil {
		text += fmt.Sprintf("\nError: %s\n", *result.Error)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, result, nil
}

// pollRunStatus handles the poll_run_status tool
func (h *MCPHandler) pollRunStatus(ctx context.Context, req *mcp.CallToolRequest, args PollRunStatusArgs) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "poll_run_status"
	metadata.Context["run_count"] = len(args.RunIDs)
	metadata.Context["timeout"] = args.Timeout
	metadata.Context["poll_interval"] = args.PollInterval
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	// Set defaults
	timeout := args.Timeout
	if timeout == 0 {
		timeout = defaultPollTimeoutSeconds
	}
	pollInterval := args.PollInterval
	if pollInterval == 0 {
		pollInterval = defaultPollIntervalMs
	}

	// Validate input
	if len(args.RunIDs) == 0 {
		return nil, nil, fmt.Errorf("runIds array cannot be empty")
	}

	startTime := time.Now()
	deadline := startTime.Add(time.Duration(timeout) * time.Second)

	var lastResult *PollRunStatusResult
	for time.Now().Before(deadline) {

		// Get status for all runs
		var runs []RunStatusResult
		completed := 0
		running := 0
		failed := 0

		for _, runID := range args.RunIDs {
			status, err := h.getRunStatusInternal(ctx, runID)
			if err != nil {
				// If run not found, skip it
				continue
			}
			runs = append(runs, *status)

			// Count statuses
			if isRunCompleted(status.Status) {
				completed++
			} else if isRunFailed(status.Status) {
				failed++
			} else {
				running++ // Treat unknown/running as running
			}
		}

		lastResult = &PollRunStatusResult{
			Runs:      runs,
			Completed: completed,
			Running:   running,
			Failed:    failed,
		}

		// If all runs are done (completed or failed), stop polling
		if running == 0 {
			break
		}

		// Sleep before next poll
		time.Sleep(time.Duration(pollInterval) * time.Millisecond)
	}

	if lastResult == nil {
		return nil, nil, fmt.Errorf("no valid runs found")
	}

	// Create summary text
	elapsed := time.Since(startTime).Round(time.Millisecond)
	text := fmt.Sprintf("Polled %d run(s) for %v\n", len(args.RunIDs), elapsed)
	text += fmt.Sprintf("Status: %d completed, %d failed, %d running\n\n",
		lastResult.Completed, lastResult.Failed, lastResult.Running)

	for _, run := range lastResult.Runs {
		text += fmt.Sprintf("--- Run %s ---\n", run.RunID)
		text += fmt.Sprintf("Function: %s\n", run.FunctionName)
		text += fmt.Sprintf("Status: %s\n", run.Status)
		text += fmt.Sprintf("Started: %s\n", run.StartedAt)
		if run.EndedAt != nil {
			text += fmt.Sprintf("Ended: %s\n", *run.EndedAt)
		}
		if run.Error != nil {
			text += fmt.Sprintf("Error: %s\n", *run.Error)
		}
		if len(run.Steps) > 0 {
			text += fmt.Sprintf("Steps: %d\n", len(run.Steps))
			lastStep := run.Steps[len(run.Steps)-1]
			text += fmt.Sprintf("Last step: %s (%s)\n", lastStep.StepName, lastStep.Status)
		}
		text += "\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, lastResult, nil
}

// invokeFunction handles the invoke_function tool
func (h *MCPHandler) invokeFunction(ctx context.Context, req *mcp.CallToolRequest, args InvokeFunctionArgs) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "invoke_function"
	metadata.Context["function_id"] = args.FunctionID
	metadata.Context["timeout"] = args.Timeout
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	// Set default timeout
	timeout := args.Timeout
	if timeout == 0 {
		timeout = defaultPollTimeoutSeconds
	}

	// Find the function by ID, slug, or name
	functions, err := h.data.GetFunctions(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get functions: %w", err)
	}

	var targetFunction *cqrs.Function
	for _, fn := range functions {
		if fn.ID.String() == args.FunctionID || fn.Slug == args.FunctionID || fn.Name == args.FunctionID {
			targetFunction = fn
			break
		}
	}

	if targetFunction == nil {
		return nil, nil, fmt.Errorf("function not found: %s", args.FunctionID)
	}

	// Create a synthetic event to trigger the function
	evt := event.Event{
		Name: fmt.Sprintf("inngest/function.invoke/%s", targetFunction.Slug),
		Data: args.Data,
		User: args.User,
	}

	startTime := time.Now()

	// Send the event to trigger the function
	evtID, err := h.events(ctx, &evt, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to invoke function: %w", err)
	}

	// Parse the event ID
	eventULID, err := ulid.Parse(evtID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse event ID: %w", err)
	}

	// Wait for the function to be triggered
	time.Sleep(h.tick * 3)

	// Get the run ID for this function
	runs, err := h.data.GetEventRuns(ctx, eventULID, consts.DevServerAccountID, consts.DevServerEnvID)
	if err != nil || len(runs) == 0 {
		return nil, nil, fmt.Errorf("function was not triggered by invoke event")
	}

	// Find the run for our target function
	var targetRunID ulid.ULID
	for _, run := range runs {
		if run.FunctionID == targetFunction.ID {
			targetRunID = run.RunID
			break
		}
	}

	if targetRunID.Compare(ulid.ULID{}) == 0 {
		return nil, nil, fmt.Errorf("no run found for function %s", args.FunctionID)
	}

	// Poll for completion
	deadline := startTime.Add(time.Duration(timeout) * time.Second)

	for {
		if time.Now().After(deadline) {
			return nil, nil, fmt.Errorf("function invocation timed out after %d seconds", timeout)
		}

		status, err := h.getRunStatusInternal(ctx, targetRunID.String())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get run status: %w", err)
		}

		// Check if run is complete
		if isRunCompleted(status.Status) {
			// Success
			duration := time.Since(startTime).Milliseconds()
			result := &InvokeFunctionResult{
				RunID:        targetRunID.String(),
				FunctionName: targetFunction.Name,
				Status:       status.Status,
				Output:       status.Output,
				Duration:     duration,
				EventID:      evtID,
			}

			if status.Error != nil {
				result.Error = status.Error
			}

			text := fmt.Sprintf("Function '%s' invoked successfully\n", targetFunction.Name)
			text += fmt.Sprintf("Status: %s\n", status.Status)
			text += fmt.Sprintf("Duration: %dms\n", duration)
			if status.Output != nil {
				text += "Function completed with output\n"
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: text,
					},
				},
			}, result, nil

		} else if isRunFailed(status.Status) {
			// Failed
			duration := time.Since(startTime).Milliseconds()
			result := &InvokeFunctionResult{
				RunID:        targetRunID.String(),
				FunctionName: targetFunction.Name,
				Status:       status.Status,
				Duration:     duration,
				EventID:      evtID,
				Error:        status.Error,
			}

			text := fmt.Sprintf("Function '%s' failed\n", targetFunction.Name)
			text += fmt.Sprintf("Status: %s\n", status.Status)
			text += fmt.Sprintf("Duration: %dms\n", duration)
			if status.Error != nil {
				text += fmt.Sprintf("Error: %s\n", *status.Error)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: text,
					},
				},
			}, result, nil

		} else {
			// Still running, continue polling
			time.Sleep(time.Duration(defaultPollIntervalMs) * time.Millisecond)
		}
	}
}

// grepDocs handles the grep_docs tool
func (h *MCPHandler) grepDocs(ctx context.Context, req *mcp.CallToolRequest, args GrepDocsArgs) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "grep_docs"
	metadata.Context["pattern_length"] = len(args.Pattern)
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	// Set default limit
	limit := args.Limit
	if limit == 0 {
		limit = 10
	}

	// Validate pattern
	if args.Pattern == "" {
		return nil, nil, fmt.Errorf("pattern cannot be empty")
	}

	// Compile the search pattern as regex
	pattern, err := regexp.Compile("(?i)" + args.Pattern)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var matches []GrepMatch

	// Walk through embedded docs filesystem
	err = walkEmbeddedDocs(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-documentation files
		if d.IsDir() || (!strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".mdx")) {
			return nil
		}

		// Read file content from cache
		content, err := h.getCachedFileContent(path)
		if err != nil {
			return nil // Skip files that can't be read
		}

		// Search for pattern in file content
		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if pattern.MatchString(line) {
				matches = append(matches, GrepMatch{
					File:    path,
					Line:    lineNum + 1, // Line numbers start at 1
					Content: strings.TrimSpace(line),
				})

				// Check if we've hit the limit
				if len(matches) >= limit {
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to search docs: %w", err)
	}

	result := &GrepDocsResult{
		Matches: matches,
		Count:   len(matches),
	}

	// Create text summary
	text := fmt.Sprintf("Found %d matches for pattern '%s':\n\n", len(matches), args.Pattern)
	for _, match := range matches {
		text += fmt.Sprintf("%s:%d: %s\n", match.File, match.Line, match.Content)
	}

	if len(matches) >= limit {
		text += fmt.Sprintf("\n(Limited to %d results)\n", limit)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, result, nil
}

// readDoc handles the read_doc tool
func (h *MCPHandler) readDoc(ctx context.Context, req *mcp.CallToolRequest, args ReadDocArgs) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "read_doc"
	metadata.Context["path"] = args.Path
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	// Validate and sanitize path
	if args.Path == "" {
		return nil, nil, fmt.Errorf("path cannot be empty")
	}

	// Prevent directory traversal
	if strings.Contains(args.Path, "..") {
		return nil, nil, fmt.Errorf("invalid path: directory traversal not allowed")
	}

	// Clean the path to use forward slashes
	cleanPath := filepath.ToSlash(args.Path)

	// Read file content from cache or embedded filesystem
	content, err := h.getCachedFileContent(cleanPath)
	if err != nil {
		return nil, nil, fmt.Errorf("file not found: %s", args.Path)
	}

	// Get file info from cache or embedded filesystem
	info, err := h.getCachedFileInfo(cleanPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, nil, fmt.Errorf("path is a directory, not a file: %s", args.Path)
	}

	result := &ReadDocResult{
		Path:    args.Path,
		Size:    info.Size(),
		Content: string(content),
	}

	// Create text summary (truncate if very long)
	text := fmt.Sprintf("File: %s\n", args.Path)
	text += fmt.Sprintf("Size: %d bytes\n\n", info.Size())

	contentStr := string(content)
	if len(contentStr) > 2000 {
		text += contentStr[:2000] + "\n\n[Content truncated - use the full content from the structured result]"
	} else {
		text += contentStr
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, result, nil
}

// listDocs handles the list_docs tool
func (h *MCPHandler) listDocs(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "list_docs"
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	// Count documents and categorize them
	categories := make(map[string]int)
	sdks := make(map[string]int)
	totalDocs := 0

	err := walkEmbeddedDocs(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Only count .md and .mdx files
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".mdx") {
			return nil
		}

		totalDocs++

		// Extract category (first directory component)
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			category := parts[0]
			categories[category]++

			// Check for SDK-specific content
			if strings.Contains(path, "typescript") || strings.Contains(path, "ts") {
				sdks["typescript"]++
			}
			if strings.Contains(path, "python") || strings.Contains(path, "py") {
				sdks["python"]++
			}
			if strings.Contains(path, "go") || strings.Contains(path, "golang") {
				sdks["go"]++
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to scan docs directory: %w", err)
	}

	result := &ListDocsResult{
		GeneratedAt: time.Now().Format(time.RFC3339),
		TotalDocs:   totalDocs,
		TotalChunks: totalDocs * 2, // Estimate chunks as 2x docs
		Categories:  categories,
		SDKs:        sdks,
	}

	// Create text summary
	text := fmt.Sprintf("Documentation Overview (Generated: %s)\n\n", result.GeneratedAt)
	text += fmt.Sprintf("Total Documents: %d\n", result.TotalDocs)
	text += fmt.Sprintf("Estimated Chunks: %d\n\n", result.TotalChunks)

	text += "Categories:\n"
	for category, count := range categories {
		text += fmt.Sprintf("  %s: %d docs\n", category, count)
	}

	text += "\nSDK Coverage:\n"
	for sdk, count := range sdks {
		text += fmt.Sprintf("  %s: %d docs\n", sdk, count)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, result, nil
}

// AddMCPRoute adds the MCP route to the dev server router
func AddMCPRoute(r chi.Router, events api.EventHandler, data cqrs.Manager, tick time.Duration) {
	r.Mount("/mcp", NewMCPHandler(events, data, tick))
}
