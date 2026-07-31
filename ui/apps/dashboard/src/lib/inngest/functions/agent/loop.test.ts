import { describe, expect, it, vi } from 'vitest';

import {
  runAgentLoop,
  type ChatCompletion,
  type QueryDraft,
  type ToolDef,
} from './loop';

function fakeStep(waitForEventResult: unknown = null) {
  return {
    run: vi.fn(async (_id: string, fn: () => unknown) => fn()),
    waitForEvent: vi.fn(async () => waitForEventResult),
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
    publish: vi.fn(async () => {}),
    runId: 'run-1',
    userId: 'user-1',
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
    expect(res.validationAttempts).toBe(0);
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

  it('applies draftPatch and publishes outside the tool step (replay-safe)', async () => {
    const published: { event: string; data: Record<string, unknown> }[] = [];
    const submit: ToolDef = {
      tool: {
        name: 'submit',
        description: '',
        input_schema: { type: 'object' },
      },
      execute: async () => ({
        observation: 'ok',
        draftPatch: { sql: 'SELECT 1', title: 'T' },
        publish: { event: 'step.completed', data: { sql: 'SELECT 1' } },
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
      publish: async (_id, event, data) => {
        published.push({ event, data });
      },
    });

    expect(res.summary).toBe('summary');
    expect(draft.sql).toBe('SELECT 1');
    expect(draft.title).toBe('T');
    expect(published).toEqual([
      { event: 'step.completed', data: { sql: 'SELECT 1' } },
    ]);
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

  describe('validate_query', () => {
    const validateTool: ToolDef = {
      tool: {
        name: 'validate_query',
        description: '',
        input_schema: { type: 'object' },
      },
      execute: async () => {
        throw new Error('handled by the loop, never called');
      },
    };

    it('publishes validation.requested and reports success from the result event', async () => {
      const step = fakeStep({
        data: {
          validationId: 'run-1-1',
          ok: true,
          columns: ['count'],
          rowCount: 42,
        },
      });
      const published: { event: string; data: Record<string, unknown> }[] = [];
      const complete = fakeComplete([
        toolResponse('validate_query', { sql: 'SELECT count() FROM runs' }),
        textResponse('done'),
      ]);
      const res = await runAgentLoop({
        ...baseArgs({ tools: [validateTool] }),
        step: step as never,
        complete,
        publish: async (_id, event, data) => {
          published.push({ event, data });
        },
      });

      expect(published[0]).toEqual({
        event: 'validation.requested',
        data: { validationId: 'run-1-1', sql: 'SELECT count() FROM runs' },
      });
      expect(step.waitForEvent).toHaveBeenCalledWith(
        'wait-validation-run-1-1',
        {
          event: 'insights-agent/validation.completed',
          timeout: '20s',
          if: 'async.data.validationId == "run-1-1" && async.data.userId == "user-1"',
        },
      );
      expect(res.validationAttempts).toBe(1);
      expect(res.validationFailures).toEqual([]);

      const secondCall = complete.mock.calls[1]?.[0] as {
        messages: { content: string }[];
      };
      expect(secondCall.messages[3]?.content).toContain('ran successfully');
    });

    it('records diagnostics as validation failures', async () => {
      const step = fakeStep({
        data: {
          validationId: 'run-1-1',
          ok: false,
          diagnostics: [
            { code: 'unknown_column', message: 'column "nope" does not exist' },
          ],
        },
      });
      const complete = fakeComplete([
        toolResponse('validate_query', { sql: 'SELECT nope FROM runs' }),
        textResponse('done'),
      ]);
      const res = await runAgentLoop({
        ...baseArgs({ tools: [validateTool] }),
        step: step as never,
        complete,
      });

      expect(res.validationAttempts).toBe(1);
      expect(res.validationFailures).toEqual([
        {
          sql: 'SELECT nope FROM runs',
          code: 'unknown_column',
          message: 'column "nope" does not exist',
        },
      ]);
    });

    it('treats a hallucinated validate_query call as an unknown tool when not offered', async () => {
      const step = fakeStep();
      const publish = vi.fn(async () => {});
      const complete = fakeComplete([
        toolResponse('validate_query', { sql: 'SELECT 1' }),
        textResponse('done'),
      ]);
      // validate_query deliberately NOT in tools (the headless path).
      const res = await runAgentLoop({
        ...baseArgs({ tools: [echoTool] }),
        step: step as never,
        complete,
        publish,
      });

      expect(publish).not.toHaveBeenCalled();
      expect(step.waitForEvent).not.toHaveBeenCalled();
      expect(res.validationAttempts).toBe(0);

      const secondCall = complete.mock.calls[1]?.[0] as {
        messages: { content: string }[];
      };
      expect(secondCall.messages[3]?.content).toContain(
        'Unknown tool: validate_query',
      );
    });

    it('degrades gracefully when no validation result arrives (timeout)', async () => {
      const step = fakeStep(null);
      const complete = fakeComplete([
        toolResponse('validate_query', { sql: 'SELECT 1' }),
        textResponse('done'),
      ]);
      const res = await runAgentLoop({
        ...baseArgs({ tools: [validateTool] }),
        step: step as never,
        complete,
      });

      expect(res.validationFailures).toEqual([]);
      const secondCall = complete.mock.calls[1]?.[0] as {
        messages: { content: string }[];
      };
      expect(secondCall.messages[3]?.content).toContain('unavailable');
    });
  });
});
