// The think→act→observe loop behind the Insights agent, ported from
// inngest-agents (packages/agent-core/src/loop.ts). Each LLM call and tool
// call is a durable Inngest step; the loop ends when the model responds with
// text and no tool calls.
//
// Wire format is OpenAI chat completions, which OpenRouter speaks natively.

export interface ToolCall {
  id: string;
  type: 'function';
  /** JSON-encoded arguments object. */
  function: { name: string; arguments: string };
}

export type AgentMessage =
  | { role: 'system' | 'user'; content: string }
  | { role: 'assistant'; content: string | null; tool_calls?: ToolCall[] }
  | { role: 'tool'; tool_call_id: string; content: string };

/** JSON Schema tool definition, translated to the wire format by the loop. */
export type ToolSpec = {
  name: string;
  description: string;
  input_schema: { type: 'object'; [k: string]: unknown };
};

interface ChatTool {
  type: 'function';
  function: {
    name: string;
    description: string;
    parameters: Record<string, unknown>;
  };
}

export interface ChatCompletion {
  choices: {
    message: {
      role: 'assistant';
      content: string | null;
      tool_calls?: ToolCall[];
    };
  }[];
  usage?: { prompt_tokens?: number; completion_tokens?: number; cost?: number };
}

export interface InsightsClientState {
  eventTypes?: string[];
  schemas?: { name: string; schema: string }[];
  currentQuery?: string;
}

export interface ToolContext {
  clientState: InsightsClientState;
}

/** Accumulates the final structured result across tool calls. */
export interface QueryDraft {
  sql?: string;
  title?: string;
  reasoning?: string;
  selectedEvents: { event_name: string; reason: string }[];
}

// Inngest only persists a step.run's return value, not closure mutations, so
// tools must RETURN their effects; the loop applies them outside the
// memoized step boundary. Mutating `draft` inside a tool would be lost on replay.
export interface ToolOutcome {
  observation: string;
  draftPatch?: Partial<QueryDraft>;
}

export interface ToolDef {
  tool: ToolSpec;
  execute: (
    input: Record<string, unknown>,
    ctx: ToolContext,
  ) => Promise<ToolOutcome>;
}

export interface AgentLoopResult {
  summary: string;
  iterations: number;
  toolCalls: number;
  tokensIn: number;
  tokensOut: number;
  // Summed usage.cost, which OpenRouter reports on every response.
  costUsd: number;
}

// The minimal slice of Inngest's step toolkit the loop uses. `run` returns
// unknown (Inngest JSON-serializes step results); call sites cast.
interface StepTools {
  run: (id: string, fn: () => unknown) => Promise<unknown>;
}

interface RunAgentLoopArgs {
  step: StepTools;
  complete: (body: {
    messages: AgentMessage[];
    tools: ChatTool[];
    max_tokens: number;
  }) => Promise<ChatCompletion>;
  system: string;
  messages: AgentMessage[];
  tools: ToolDef[];
  ctx: ToolContext;
  draft: QueryDraft;
  maxIterations?: number;
}

const FINAL_ITERATION_NUDGE =
  '[SYSTEM: This is your final iteration. If you have a query ready, call submit_query then summarize; otherwise answer or ask your clarifying question now. Do not call any other tools.]';

// Degrades to no input rather than throwing, so a malformed call surfaces as the
// tool's own validation error instead of burning the run's retries.
export function parseToolArguments(raw: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(raw || '{}');
    return typeof parsed === 'object' && parsed !== null ? parsed : {};
  } catch {
    return {};
  }
}

export async function runAgentLoop(
  args: RunAgentLoopArgs,
): Promise<AgentLoopResult> {
  const { step, complete, system, tools, ctx, draft } = args;
  const maxIterations = args.maxIterations ?? 12;
  const messages: AgentMessage[] = [
    { role: 'system', content: system },
    ...args.messages,
  ];
  const registry = new Map(tools.map((t) => [t.tool.name, t]));
  const chatTools: ChatTool[] = tools.map((t) => ({
    type: 'function',
    function: {
      name: t.tool.name,
      description: t.tool.description,
      parameters: t.tool.input_schema,
    },
  }));

  let iterations = 0;
  let toolCalls = 0;
  let summary = '';
  let tokensIn = 0;
  let tokensOut = 0;
  let costUsd = 0;

  while (iterations < maxIterations) {
    iterations++;

    // Nudge the model to wrap up on the last iteration. The nudge goes only
    // into this call's message list, never into the running conversation.
    const turnMessages = [...messages];
    if (iterations === maxIterations) {
      turnMessages.push({ role: 'user', content: FINAL_ITERATION_NUDGE });
    }

    // No tool_choice: the model freely picks between a tool call and final text.
    // A thrown LLM call fails this attempt and Inngest retries the step; if all
    // retries exhaust, the run fails and onFailure reports it to the UI.
    const response = (await step.run(`think-${iterations}`, () =>
      complete({
        messages: turnMessages,
        tools: chatTools,
        max_tokens: 4096,
      }),
    )) as ChatCompletion;

    tokensIn += response.usage?.prompt_tokens ?? 0;
    tokensOut += response.usage?.completion_tokens ?? 0;
    costUsd += response.usage?.cost ?? 0;

    const message = response.choices[0]?.message;
    const requestedCalls = message?.tool_calls ?? [];
    const text = message?.content ?? '';

    if (text) {
      summary = text;
    }
    if (requestedCalls.length === 0) {
      break;
    }

    // The assistant turn must round-trip verbatim so tool_call ids match.
    messages.push({
      role: 'assistant',
      content: message?.content ?? null,
      tool_calls: requestedCalls,
    });

    for (const call of requestedCalls) {
      toolCalls++;
      const input = parseToolArguments(call.function.arguments);
      const name = call.function.name;

      const def = registry.get(name);
      const outcome: ToolOutcome = def
        ? ((await step.run(`tool-${name}-${iterations}-${toolCalls}`, () =>
            def.execute(input, ctx),
          )) as ToolOutcome)
        : { observation: `Unknown tool: ${name}` };

      // The patch is applied here, outside the memoized step (see ToolOutcome).
      if (outcome.draftPatch) Object.assign(draft, outcome.draftPatch);

      messages.push({
        role: 'tool',
        tool_call_id: call.id,
        content: outcome.observation,
      });
    }
  }

  return {
    summary,
    iterations,
    toolCalls,
    tokensIn,
    tokensOut,
    costUsd,
  };
}
