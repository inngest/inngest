import type Anthropic from '@anthropic-ai/sdk';

// The think→act→observe loop behind the Insights agent, ported from
// inngest-agents (packages/agent-core/src/loop.ts). Each LLM call and tool
// call is a durable Inngest step; the loop ends when the model responds with
// text and no tool calls.

export const VALIDATE_QUERY = 'validate_query';
export const VALIDATION_COMPLETED_EVENT = 'insights-agent/validation.completed';

export type AgentMessage = Anthropic.MessageParam;

export type AnthropicTool = {
  name: string;
  description: string;
  input_schema: { type: 'object'; [k: string]: unknown };
};

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
  publish?: { event: string; data: Record<string, unknown> };
}

export interface ToolDef {
  tool: AnthropicTool;
  execute: (
    input: Record<string, unknown>,
    ctx: ToolContext,
  ) => Promise<ToolOutcome>;
}

/** What the browser reports back after running the SQL on the agent's behalf. */
export interface ValidationResult {
  validationId: string;
  ok: boolean;
  columns?: string[];
  rowCount?: number;
  diagnostics?: { code?: string; message: string }[];
}

export interface ValidationFailure {
  sql: string;
  code: string;
  message: string;
}

export interface AgentLoopResult {
  summary: string;
  iterations: number;
  toolCalls: number;
  tokensIn: number;
  tokensOut: number;
  validationAttempts: number;
  validationFailures: ValidationFailure[];
}

// The minimal slice of Inngest's step toolkit the loop uses. `run` returns
// unknown (Inngest JSON-serializes step results); call sites cast.
interface StepTools {
  run: (id: string, fn: () => unknown) => Promise<unknown>;
  waitForEvent: (
    id: string,
    opts: { event: string; timeout: string; if: string },
  ) => Promise<{ data: Record<string, unknown> } | null>;
}

interface RunAgentLoopArgs {
  step: StepTools;
  client: Anthropic;
  model: string;
  system: string;
  messages: AgentMessage[];
  tools: ToolDef[];
  ctx: ToolContext;
  draft: QueryDraft;
  publish: (
    id: string,
    event: string,
    data: Record<string, unknown>,
  ) => Promise<unknown>;
  runId: string;
  // Clerk id of the user whose browser may answer validate_query round trips.
  // The wait condition pins on it so only that user's authenticated result can
  // complete the validation. Empty string fails closed (nothing matches).
  userId: string;
  maxIterations?: number;
}

const FINAL_ITERATION_NUDGE =
  '[SYSTEM: This is your final iteration. If you have a query ready, call submit_query then summarize; otherwise answer or ask your clarifying question now. Do not call any other tools.]';

export async function runAgentLoop(
  args: RunAgentLoopArgs,
): Promise<AgentLoopResult> {
  const {
    step,
    client,
    model,
    system,
    tools,
    ctx,
    draft,
    publish,
    runId,
    userId,
  } = args;
  const maxIterations = args.maxIterations ?? 12;
  const messages: AgentMessage[] = [...args.messages];
  const registry = new Map(tools.map((t) => [t.tool.name, t]));

  let iterations = 0;
  let toolCalls = 0;
  let summary = '';
  let tokensIn = 0;
  let tokensOut = 0;
  let validationAttempts = 0;
  const validationFailures: ValidationFailure[] = [];

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
      client.messages.create({
        model,
        max_tokens: 4096,
        system,
        messages: turnMessages,
        tools: tools.map((t) => t.tool) as Anthropic.Messages.Tool[],
      }),
    )) as Anthropic.Message;

    tokensIn += response.usage?.input_tokens ?? 0;
    tokensOut += response.usage?.output_tokens ?? 0;

    const toolUses = response.content.filter(
      (block): block is Anthropic.ToolUseBlock => block.type === 'tool_use',
    );
    const text = response.content
      .filter((block): block is Anthropic.TextBlock => block.type === 'text')
      .map((block) => block.text)
      .join('\n');

    if (toolUses.length === 0) {
      summary = text;
      break;
    }

    // The assistant turn must round-trip verbatim so tool_use ids match.
    messages.push({ role: 'assistant', content: response.content });

    const toolResults: Anthropic.ToolResultBlockParam[] = [];
    for (const toolUse of toolUses) {
      toolCalls++;
      const input = (toolUse.input ?? {}) as Record<string, unknown>;

      let outcome: ToolOutcome;
      if (toolUse.name === VALIDATE_QUERY && registry.has(VALIDATE_QUERY)) {
        // Validation uses durable primitives that can't nest inside step.run;
        // the registry check drops hallucinated calls when the tool wasn't offered.
        validationAttempts++;
        outcome = await validateQuery({
          sql: String(input.sql ?? ''),
          validationId: `${runId}-${toolCalls}`,
          userId,
          step,
          publish,
          iterations,
          toolCalls,
          validationFailures,
        });
      } else {
        const def = registry.get(toolUse.name);
        outcome = def
          ? ((await step.run(
              `tool-${toolUse.name}-${iterations}-${toolCalls}`,
              () => def.execute(input, ctx),
            )) as ToolOutcome)
          : { observation: `Unknown tool: ${toolUse.name}` };
      }

      // Effects are applied here, outside the memoized step (see ToolOutcome).
      if (outcome.draftPatch) Object.assign(draft, outcome.draftPatch);
      if (outcome.publish) {
        await publish(
          `publish-${outcome.publish.event}-${iterations}-${toolCalls}`,
          outcome.publish.event,
          outcome.publish.data,
        );
      }

      toolResults.push({
        type: 'tool_result',
        tool_use_id: toolUse.id,
        content: outcome.observation,
      });
    }
    messages.push({ role: 'user', content: toolResults });
  }

  return {
    summary,
    iterations,
    toolCalls,
    tokensIn,
    tokensOut,
    validationAttempts,
    validationFailures,
  };
}

// Ask the user's browser (subscribed to the agent stream) to run the SQL with
// its own credentials, and wait for the result event. See InsightsChatProvider
// and /api/chat-validate for the other half of the round trip.
async function validateQuery(args: {
  sql: string;
  validationId: string;
  userId: string;
  step: StepTools;
  publish: RunAgentLoopArgs['publish'];
  iterations: number;
  toolCalls: number;
  validationFailures: ValidationFailure[];
}): Promise<ToolOutcome> {
  const { sql, validationId, userId, step, publish } = args;

  await publish(
    `publish-validation.requested-${args.iterations}-${args.toolCalls}`,
    'validation.requested',
    { validationId, sql },
  );

  // userId is stamped server-side by /api/chat-validate from the poster's
  // Clerk session, so only the initiating user can complete this validation.
  const completed = await step.waitForEvent(`wait-validation-${validationId}`, {
    event: VALIDATION_COMPLETED_EVENT,
    timeout: '20s',
    if: `async.data.validationId == "${validationId}" && async.data.userId == "${userId}"`,
  });

  if (!completed) {
    return {
      observation:
        'Validation is unavailable right now (no result within 20s). Proceed without it and do not call validate_query again this run.',
    };
  }

  const result = completed.data as unknown as ValidationResult;
  if (result.ok) {
    const columns = (result.columns ?? []).join(', ');
    const emptyNote =
      result.rowCount === 0
        ? ' The query is valid but returned 0 rows — consider whether the filters are too narrow.'
        : '';
    return {
      observation: `Query ran successfully. Columns: ${columns}. Rows: ${result.rowCount}.${emptyNote}`,
    };
  }

  const diagnostics = result.diagnostics ?? [];
  for (const d of diagnostics) {
    args.validationFailures.push({
      sql,
      code: d.code || 'error',
      message: d.message,
    });
  }
  const details = diagnostics
    .map((d) => `- [${d.code || 'error'}] ${d.message}`)
    .join('\n');
  return {
    observation: `Query failed validation:\n${details}\nFix the SQL and validate again.`,
  };
}
