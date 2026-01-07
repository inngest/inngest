# Agent Observability Pattern

## System Prompt Hydration Observability

### Problem

When using `createAgent` from `@inngest/agent-kit`, the `system` function runs inside the agent-kit framework, not within Inngest steps. This means system prompt hydration (where we render templates with dynamic data) doesn't appear as a separate step in Inngest traces.

### Solution

Store the system prompt hydration context in the network state. This data becomes part of the agent's observable state and can be inspected through:

1. Agent-kit's streaming events
2. Network state inspection
3. Debugging tools

### Implementation Example

See [event-matcher/index.ts](./event-matcher/index.ts) for the reference implementation:

```typescript
export const eventMatcherAgent = createAgent<InsightsAgentState>({
  name: 'Insights Event Matcher',
  system: async ({ network }): Promise<string> => {
    const events = network?.state.data.eventTypes || [];
    const sample = events.slice(0, 500);

    // Prepare context for system prompt hydration
    const promptContext = {
      totalEvents: events.length,
      hasEvents: sample.length > 0,
      eventsList: sample.join('\n'),
      maxEvents: 500,
    };

    // Store prompt context in network state for observability
    // This will be captured in agent-kit's streaming events
    if (network?.state.data) {
      network.state.data.eventMatcherPromptContext = promptContext;
    }

    return Mustache.render(systemPrompt, promptContext);
  },
  // ... rest of agent config
});
```

### Type Safety

Add the context type to `InsightsAgentState` in [types.ts](./types.ts):

```typescript
export type InsightsAgentState = StateData & {
  // ... other fields

  // Observability: System prompt hydration context for agents
  eventMatcherPromptContext?: {
    totalEvents: number;
    hasEvents: boolean;
    eventsList: string;
    maxEvents: number;
  };
};
```

### Benefits

1. **Debugging**: Inspect what data was used to hydrate system prompts
2. **Monitoring**: Track prompt context across agent executions
3. **Reproducibility**: Recreate exact prompt conditions from state snapshots
4. **No framework changes**: Works within existing agent-kit architecture

### Accessing the Data

The prompt context is available in multiple places:

1. **Inngest Step Output**: The `capture-observability-data` step captures the network state AFTER execution completes

   ```typescript
   // In run-network.ts
   // network.run() returns a NetworkRun instance with the mutated state
   const networkRun = await network.run(userMessage, { streaming: { ... } });

   // Capture state in a separate step (runs after network completes)
   // CRITICAL: Use networkRun.state, not network.state
   await step.run('capture-observability-data', async () => {
     return {
       promptContext: networkRun.state.data.eventMatcherPromptContext,
       selectedEvents: networkRun.state.data.selectedEvents,
       sql: networkRun.state.data.sql,
     };
   });
   ```

   **Important Notes**:

   - The observability step runs AFTER `network.run()` completes to avoid blocking real-time streaming
   - **CRITICAL**: `network.run()` returns a `NetworkRun` instance - all state mutations happen on this returned instance, not the original `Network` instance. You must use `networkRun.state`, not `network.state`

2. **Network State**: During agent execution
   - **Network state**: `network.state.data.eventMatcherPromptContext`
   - **Tool handlers**: Accessible via `network` parameter in tool handlers
   - **Downstream agents**: Available to subsequent agents in the network

### Viewing in Inngest UI

When you inspect a function run in Inngest, you'll see a step called `capture-observability-data` with output structured to show the user's request followed by each agent's prompt context and output:

```json
{
  "userPrompt": "Show me all successful payments",
  "timestamp": "2025-01-06T10:30:00.000Z",

  "agents": {
    "eventMatcher": {
      "promptContext": {
        "totalEvents": 150,
        "hasEvents": true,
        "maxEvents": 500,
        "eventsListLength": 1775,
        "eventsListPreview": "app/account.created\nbilling/payment.succeeded\n..."
      },
      "output": {
        "selectedEvents": [
          {
            "event_name": "billing/payment.succeeded",
            "reason": "Direct match for succeeded payments..."
          }
        ],
        "selectionReason": "Selected by the LLM based on the user's query."
      }
    },

    "queryWriter": {
      "promptContext": {
        "selectedEventsCount": 1,
        "selectedEventNames": ["billing/payment.succeeded"],
        "schemasCount": 1,
        "schemaNames": ["billing/payment.succeeded"]
      },
      "output": {
        "sql": "SELECT * FROM events WHERE event_name = 'billing/payment.succeeded'",
        "title": "Successful Payments Query",
        "reasoning": "Filters events to show only successful payment transactions"
      }
    },

    "summarizer": {
      "promptContext": {
        "selectedEventsCount": 1,
        "selectedEventNames": ["billing/payment.succeeded"],
        "hasSql": true
      },
      "output": "This query retrieves all successful payment events..."
    }
  },

  "debug": {
    "stateKeys": ["userId", "eventTypes", "schemas", ...],
    "agentsRun": ["Insights Event Matcher", "Insights Query Writer", "Insights Summarizer"],
    "resultsCount": 3
  }
}
```

### Data Structure Benefits

1. **User Context First**: The `userPrompt` shows what the user asked for
2. **Per-Agent Visibility**: Each agent has its own section with:
   - `promptContext`: What data was available when hydrating the system prompt
   - `output`: What the agent produced
3. **Full Traceability**: You can see exactly how data flows through the network:
   - Event Matcher: Which events were considered → which were selected
   - Query Writer: Which events/schemas were used → what SQL was generated
   - Summarizer: What context was available → what summary was created
4. **Debug Info**: State keys and execution metadata for troubleshooting

**Note**: Long strings are truncated for readability:

- `eventsListPreview`: First 500 characters of the full event list
- Full data is available in the network state during execution

### Alternative Approaches Considered

1. ❌ **Direct step.run() in system function**: Not possible - `system` function doesn't receive `step` context
2. ❌ **Modify agent-kit**: Would require upstream framework changes
3. ✅ **Network state storage**: Current approach - works within existing architecture
4. ✅ **Streaming events**: Agent-kit already emits detailed events for observability

### Future Enhancements

If agent-kit adds built-in support for step context in the future, we could emit custom step events:

```typescript
// Hypothetical future API if agent-kit adds step support
system: async ({ network, step }) => {
  const promptContext = await step.run('hydrate-system-prompt', async () => {
    // hydration logic here
    return { /* context */ };
  });

  return Mustache.render(systemPrompt, promptContext);
}
```

Until then, storing in network state provides the observability we need.
