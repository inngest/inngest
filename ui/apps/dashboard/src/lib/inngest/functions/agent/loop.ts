import type Anthropic from '@anthropic-ai/sdk';

export type AgentMessage = Anthropic.MessageParam;
export type AnthropicTool = {
  name: string;
  description: string;
  input_schema: { type: 'object'; [k: string]: unknown };
};

/** Mutable accumulator for the final structured result. Never mutated inside a step.run. */
export interface QueryDraft {
  sql?: string;
  title?: string;
  reasoning?: string;
  tables?: string[];
  selectedEvents: { event_name: string; reason: string }[];
}

export interface InsightsClientState {
  eventTypes?: string[];
  schemas?: { name: string; schema: string }[];
  currentQuery?: string;
}

export interface ToolContext {
  clientState: InsightsClientState;
}

/** Serializable result of a tool — applied to draft / published OUTSIDE step memoization. */
export interface ToolOutcome {
  observation: string;
  draftPatch?: Partial<QueryDraft>;
  publish?: { event: string; data: Record<string, unknown> };
}

export interface ToolDef {
  tool: AnthropicTool;
  execute: (input: any, ctx: ToolContext) => Promise<ToolOutcome>;
}

export interface AgentLoopResult {
  summary: string;
  draft: QueryDraft;
  iterations: number;
  toolCalls: number;
  tokensIn: number;
  tokensOut: number;
}

interface RunAgentLoopArgs {
  step: any;
  client: Anthropic;
  model: string;
  maxTokens?: number;
  system: string;
  messages: AgentMessage[];
  tools: ToolDef[];
  ctx: ToolContext;
  draft: QueryDraft;
  publish: (
    id: string,
    event: string,
    data: Record<string, unknown>,
  ) => Promise<void>;
  maxIterations: number;
}

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
    maxIterations,
  } = args;
  const maxTokens = args.maxTokens ?? 4096;
  const messages: AgentMessage[] = [...args.messages];
  const toolDefs = tools.map((t) => t.tool);
  const registry = new Map(tools.map((t) => [t.tool.name, t]));

  let iterations = 0;
  let toolCalls = 0;
  let summary = '';
  let tokensIn = 0;
  let tokensOut = 0;

  while (iterations < maxIterations) {
    iterations++;

    const budget =
      iterations >= maxIterations - 1
        ? '\n\n[SYSTEM: This is the final iteration. If you can answer, call submit_query then summarize. If the request is ambiguous, ask your clarifying question now. Do not request more tools.]'
        : '';
    const turnMessages = budget
      ? [...messages, { role: 'user' as const, content: budget }]
      : messages;

    // Direct Anthropic SDK call, made durable by the surrounding step.run.
    // No tool_choice → default 'auto' so the model can pick a tool or emit final text.
    const response: Anthropic.Message = await step.run(
      `think-${iterations}`,
      () =>
        client.messages.create({
          model,
          max_tokens: maxTokens,
          system,
          messages: turnMessages,
          tools: toolDefs,
        }),
    );

    tokensIn += response.usage?.input_tokens ?? 0;
    tokensOut += response.usage?.output_tokens ?? 0;

    const content = response.content;
    const toolUses = content.filter(
      (b): b is Anthropic.ToolUseBlock => b.type === 'tool_use',
    );
    const text = content
      .filter((b): b is Anthropic.TextBlock => b.type === 'text')
      .map((b) => b.text)
      .join('\n');

    if (toolUses.length === 0) {
      summary = text;
      break;
    }

    messages.push({ role: 'assistant', content });

    const toolResults: Anthropic.ToolResultBlockParam[] = [];
    for (const tu of toolUses) {
      toolCalls++;
      const def = registry.get(tu.name);
      const outcome: ToolOutcome = def
        ? await step.run(`tool-${tu.name}-${iterations}-${toolCalls}`, () =>
            def.execute(tu.input, ctx),
          )
        : { observation: `Unknown tool: ${tu.name}` };

      // Apply state changes outside the memoized step (replay-safe).
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
        tool_use_id: tu.id,
        content: outcome.observation,
      });
    }
    messages.push({ role: 'user', content: toolResults });
  }

  return { summary, draft, iterations, toolCalls, tokensIn, tokensOut };
}
