package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/oklog/ulid/v2"
)

const (
	// Default polling configuration
	defaultPollTimeoutSeconds = 30
	defaultPollIntervalMs     = 1000

	// Event processing delay
	eventProcessingDelay = 500 * time.Millisecond

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
}

// convertToDataMap converts various input types to a map suitable for event data
func convertToDataMap(input any) map[string]any {
	if input == nil {
		return nil
	}

	switch data := input.(type) {
	case map[string]any:
		return data
	case string:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(data), &parsed); err == nil {
			return parsed
		}
		return map[string]any{"value": data}
	default:
		return map[string]any{"value": input}
	}
}

// convertToUserMap converts various input types to a map suitable for user data
func convertToUserMap(input any) map[string]any {
	if input == nil {
		return nil
	}

	switch user := input.(type) {
	case map[string]any:
		return user
	case string:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(user), &parsed); err == nil {
			return parsed
		}
		return map[string]any{"id": user}
	default:
		return map[string]any{"id": input}
	}
}


// NewMCPHandler creates a new MCP handler for the dev server
func NewMCPHandler(events api.EventHandler, data cqrs.Manager) http.Handler {
	h := &MCPHandler{
		events: events,
		data:   data,
	}

	// Create a streamable HTTP handler that returns the same server for all requests
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return h.createMCPServer()
	}, &mcp.StreamableHTTPOptions{
		JSONResponse: true, // Use JSON responses for better compatibility
	})
}

// createMCPServer creates an MCP server instance with dev server tools
func (h *MCPHandler) createMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    mcpName,
		Version: mcpVersion,
		Title:   mcpTitle,
	}, nil)

	// Add the send event tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "send_event",
		Description: "Send an event to the Inngest dev server which will trigger any functions listening to that event. Returns event ID and run IDs of triggered functions. Parameters: name (required string - the event name like 'test/hello.world'), data (optional JSON object - the event data), user (optional JSON object - user context), eventIdSeed (optional string for deterministic event IDs)",
	}, h.sendEvent)

	// Add list functions tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_functions",
		Description: "List all registered functions in the dev server",
	}, h.listFunctions)

	// Add get run status tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_run_status",
		Description: "Get detailed status and trace information for a specific function run. Parameters: runId (required string - the run ID returned from send_event or found in logs)",
	}, h.getRunStatus)

	// Add poll run status tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "poll_run_status",
		Description: "Poll multiple function runs until they complete or timeout. Returns detailed status for all runs. Parameters: runIds (required array of strings - run IDs to poll), timeout (optional int - seconds to poll, default 30), pollInterval (optional int - milliseconds between polls, default 1000)",
	}, h.pollRunStatus)

	return server
}

// SendEventArgs represents the arguments for sending an event
type SendEventArgs struct {
	Name        string `json:"name"`
	Data        any    `json:"data,omitempty"`
	User        any    `json:"user,omitempty"`
	EventIDSeed string `json:"eventIdSeed,omitempty"`
}

// SendEventResult represents the result of sending an event
type SendEventResult struct {
	EventID string   `json:"eventId"`
	RunIDs  []string `json:"runIds,omitempty"`
	Status  string   `json:"status"`
	Message string   `json:"message"`
}

// sendEvent handles the send_event tool
func (h *MCPHandler) sendEvent(ctx context.Context, req *mcp.CallToolRequest, args SendEventArgs) (*mcp.CallToolResult, any, error) {
	// Track MCP tool usage
	metadata := tel.NewMetadata(ctx)
	metadata.Context["tool"] = "send_event"
	metadata.Context["event_name"] = args.Name
	if args.EventIDSeed != "" {
		metadata.Context["has_event_id_seed"] = true
	}
	tel.SendEvent(ctx, "cli/mcp.tool.executed", metadata)

	// Create the event
	evt := event.Event{
		Name: args.Name,
	}

	// Set event data and user using helper functions
	evt.Data = convertToDataMap(args.Data)
	evt.User = convertToUserMap(args.User)

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
	time.Sleep(eventProcessingDelay)

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

	// Create response text
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

	// Prepare the result
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
		var outputData map[string]interface{}
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
	for {
		// Check timeout
		if time.Now().After(deadline) {
			break
		}

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
			switch status.Status {
			case statusCompleted, statusSkipped:
				completed++
			case statusFailed, statusCancelled, statusOverflowed:
				failed++
			default:
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

// AddMCPRoute adds the MCP route to the dev server router
func AddMCPRoute(r chi.Router, events api.EventHandler, data cqrs.Manager) {
	r.Mount("/mcp", NewMCPHandler(events, data))
}
