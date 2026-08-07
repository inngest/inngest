import { describe, expect, it, vi } from 'vitest';

import {
  runAgentLoop,
  type ChatCompletion,
  type QueryDraft,
  type ToolDef,
} from './loop';

function fakeStep() {
  return {
    run: vi.fn(async (_id: string, fn: () => unknown) => fn()),
  };
}

// Replays canned completions in order, recording each request body.
function fakeComplete(responses: unknown[]) {
  let i = 0;
  return vi.fn(
    (_body: unknown): Promise<ChatCompletion> =>
      Promise.resolve(responses[i++] as ChatCompletion),
  );
}

const echoTool: ToolDef = {
  tool: { name: 'echo', description: 'echo', input_schema: { type: 'object' } },
  execute: async (input) => ({ observation: `echoed:${input.msg}` }),
};

function baseArgs(overrides: Record<string, unknown> = {}) {
  return {
    step: fakeStep() as never,
    system: 'system prompt',
    messages: [{ role: 'user' as const, content: 'hi' }],
    tools: [echoTool],
    ctx: { clientState: {} },
    draft: { selectedEvents: [] } as QueryDraft,
    ...overrides,
  };
}

function textResponse(text: string) {
  return {
    choices: [{ message: { role: 'assistant', content: text } }],
    usage: { prompt_tokens: 10, completion_tokens: 5 },
  };
}

function toolResponse(name: string, input: Record<string, unknown>, id = 't1') {
  return {
    choices: [
      {
        message: {
          role: 'assistant',
          content: null,
          tool_calls: [
            {
              id,
              type: 'function',
              function: { name, arguments: JSON.stringify(input) },
            },
          ],
        },
      },
    ],
    usage: { prompt_tokens: 10, completion_tokens: 5 },
  };
}

describe('runAgentLoop', () => {
  it('returns a text-only response as the summary without calling tools', async () => {
    const complete = fakeComplete([textResponse('done summary')]);
    const res = await runAgentLoop({ ...baseArgs(), complete });

    expect(res.summary).toBe('done summary');
    expect(res.iterations).toBe(1);
    expect(res.toolCalls).toBe(0);
    expect(res.tokensIn).toBe(10);
    expect(res.tokensOut).toBe(5);
  });

  it('executes tool calls then finishes on a text-only turn', async () => {
    const complete = fakeComplete([
      toolResponse('echo', { msg: 'x' }),
      textResponse('final'),
    ]);
    const res = await runAgentLoop({ ...baseArgs(), complete });

    expect(res.toolCalls).toBe(1);
    expect(res.iterations).toBe(2);
    expect(res.summary).toBe('final');

    // The second LLM call must carry the assistant turn and the tool result.
    // Messages: [system, user, assistant(tool_calls), tool].
    const secondCall = complete.mock.calls[1]?.[0] as {
      messages: { role: string; content: unknown; tool_call_id?: string }[];
    };
    expect(secondCall.messages).toHaveLength(4);
    expect(secondCall.messages[2]?.role).toBe('assistant');
    expect(secondCall.messages[3]).toEqual({
      role: 'tool',
      tool_call_id: 't1',
      content: 'echoed:x',
    });
  });

  it('applies draftPatch outside the tool step (replay-safe)', async () => {
    const submit: ToolDef = {
      tool: {
        name: 'submit',
        description: '',
        input_schema: { type: 'object' },
      },
      execute: async () => ({
        observation: 'ok',
        draftPatch: { sql: 'SELECT 1', title: 'T' },
      }),
    };
    const complete = fakeComplete([
      toolResponse('submit', {}),
      textResponse('summary'),
    ]);
    const draft: QueryDraft = { selectedEvents: [] };
    const res = await runAgentLoop({
      ...baseArgs({ tools: [submit], draft }),
      complete,
    });

    expect(res.summary).toBe('summary');
    expect(draft.sql).toBe('SELECT 1');
    expect(draft.title).toBe('T');
  });

  it('stops at maxIterations and nudges the model on the final call', async () => {
    const responses = Array.from({ length: 3 }, (_, i) =>
      toolResponse('echo', { msg: 'x' }, `t${i}`),
    );
    const complete = fakeComplete(responses);
    const res = await runAgentLoop({
      ...baseArgs(),
      complete,
      maxIterations: 3,
    });

    expect(res.iterations).toBe(3);
    expect(res.summary).toBe('');

    const lastCall = complete.mock.calls[2]?.[0] as {
      messages: { content: unknown }[];
    };
    const lastMessage = lastCall.messages[lastCall.messages.length - 1];
    expect(lastMessage?.content).toContain('final iteration');
  });

  it('keeps text that accompanies a tool call on the final iteration', async () => {
    const complete = fakeComplete([
      {
        choices: [
          {
            message: {
              role: 'assistant',
              content: 'Here is your query.',
              tool_calls: [
                {
                  id: 't1',
                  type: 'function',
                  function: {
                    name: 'echo',
                    arguments: JSON.stringify({ msg: 'x' }),
                  },
                },
              ],
            },
          },
        ],
        usage: { prompt_tokens: 10, completion_tokens: 5 },
      },
    ]);
    const res = await runAgentLoop({
      ...baseArgs(),
      complete,
      maxIterations: 1,
    });

    expect(res.summary).toBe('Here is your query.');
  });

  it('reports an unknown tool call back to the model', async () => {
    const complete = fakeComplete([
      toolResponse('validate_query', { sql: 'SELECT 1' }),
      textResponse('done'),
    ]);
    const res = await runAgentLoop({ ...baseArgs(), complete });

    expect(res.toolCalls).toBe(1);

    const secondCall = complete.mock.calls[1]?.[0] as {
      messages: { content: string }[];
    };
    expect(secondCall.messages[3]?.content).toContain(
      'Unknown tool: validate_query',
    );
  });
});
