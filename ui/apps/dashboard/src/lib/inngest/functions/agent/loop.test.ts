import { describe, it, expect, vi } from 'vitest';

import { runAgentLoop, type ToolDef } from './loop';

// step.run just invokes the thunk; the think call's LLM response comes from the fake client.
function fakeStep() {
  return {
    run: vi.fn(async (_id: string, fn: () => unknown) => fn()),
    realtime: { publish: vi.fn(async () => {}) },
  };
}

// messages.create() returns canned responses in order.
function fakeClient(responses: any[]) {
  let i = 0;
  return { messages: { create: vi.fn(async () => responses[i++]) } };
}

const echoTool: ToolDef = {
  tool: { name: 'echo', description: 'echo', input_schema: { type: 'object' } },
  execute: async (input: any) => ({ observation: `echoed:${input.msg}` }),
};

const base = {
  model: 'claude-sonnet-4-5',
  system: 's',
  messages: [{ role: 'user' as const, content: 'hi' }],
  ctx: { clientState: {} as any },
  publish: async () => {},
  maxIterations: 5,
};

describe('runAgentLoop', () => {
  it('returns a text-only response as the summary without calling tools', async () => {
    const client = fakeClient([
      { content: [{ type: 'text', text: 'done summary' }], usage: {} },
    ]);
    const res = await runAgentLoop({
      ...base,
      step: fakeStep() as any,
      client: client as any,
      tools: [echoTool],
      draft: { selectedEvents: [] },
    });
    expect(res.summary).toBe('done summary');
    expect(res.iterations).toBe(1);
    expect(res.toolCalls).toBe(0);
  });

  it('executes tool calls then finishes on a text-only turn', async () => {
    const client = fakeClient([
      {
        content: [
          { type: 'tool_use', id: 't1', name: 'echo', input: { msg: 'x' } },
        ],
        usage: {},
      },
      { content: [{ type: 'text', text: 'final' }], usage: {} },
    ]);
    const res = await runAgentLoop({
      ...base,
      step: fakeStep() as any,
      client: client as any,
      tools: [echoTool],
      draft: { selectedEvents: [] },
    });
    expect(res.toolCalls).toBe(1);
    expect(res.iterations).toBe(2);
    expect(res.summary).toBe('final');
    expect(client.messages.create).toHaveBeenCalledTimes(2);
  });

  it('applies draftPatch returned by a tool (replay-safe) and publishes its event', async () => {
    const published: any[] = [];
    const submit: ToolDef = {
      tool: {
        name: 'submit',
        description: '',
        input_schema: { type: 'object' },
      },
      execute: async () => ({
        observation: 'ok',
        draftPatch: { sql: 'SELECT 1', title: 'T', tables: ['runs'] },
        publish: {
          event: 'step.completed',
          data: { step: 'query-writer', sql: 'SELECT 1' },
        },
      }),
    };
    const client = fakeClient([
      {
        content: [{ type: 'tool_use', id: 't1', name: 'submit', input: {} }],
        usage: {},
      },
      { content: [{ type: 'text', text: 'summary' }], usage: {} },
    ]);
    const res = await runAgentLoop({
      ...base,
      step: fakeStep() as any,
      client: client as any,
      tools: [submit],
      draft: { selectedEvents: [] },
      publish: async (_id, event, data) => {
        published.push({ event, data });
      },
    });
    expect(res.draft.sql).toBe('SELECT 1');
    expect(res.draft.tables).toEqual(['runs']);
    expect(published[0].event).toBe('step.completed');
  });

  it('clarification: text-only with no submit leaves draft.sql empty', async () => {
    const client = fakeClient([
      {
        content: [
          { type: 'text', text: 'Did you mean failed runs or failed steps?' },
        ],
        usage: {},
      },
    ]);
    const res = await runAgentLoop({
      ...base,
      step: fakeStep() as any,
      client: client as any,
      tools: [echoTool],
      draft: { selectedEvents: [] },
    });
    expect(res.summary).toMatch(/failed runs or failed steps/);
    expect(res.draft.sql).toBeUndefined();
  });

  it('stops at maxIterations even if the model keeps calling tools', async () => {
    const client = fakeClient(
      Array.from({ length: 10 }, () => ({
        content: [
          { type: 'tool_use', id: 't', name: 'echo', input: { msg: 'x' } },
        ],
        usage: {},
      })),
    );
    const res = await runAgentLoop({
      ...base,
      step: fakeStep() as any,
      client: client as any,
      tools: [echoTool],
      draft: { selectedEvents: [] },
      maxIterations: 3,
    });
    expect(res.iterations).toBe(3);
  });
});
