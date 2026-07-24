import { describe, expect, it } from 'vitest';

import type { ToolContext } from './loop';
import { findEventsTool, getEventSchemasTool, submitQueryTool } from './tools';

const ctx: ToolContext = {
  clientState: {
    eventTypes: [
      'app/user.created',
      'app/user.deleted',
      'billing/charge.succeeded',
    ],
    schemas: [{ name: 'app/user.created', schema: '{ "id": "string" }' }],
  },
};

describe('findEventsTool', () => {
  it('lists all events with no search term', async () => {
    const out = await findEventsTool.execute({}, ctx);
    expect(out.observation).toContain('app/user.created');
    expect(out.observation).toContain('billing/charge.succeeded');
  });

  it('filters events case-insensitively', async () => {
    const out = await findEventsTool.execute({ search: 'USER' }, ctx);
    expect(out.observation).toContain('app/user.created');
    expect(out.observation).not.toContain('billing/charge.succeeded');
  });

  it('reports when nothing matches', async () => {
    const out = await findEventsTool.execute({ search: 'zzz' }, ctx);
    expect(out.observation).toBe('No matching event types.');
  });
});

describe('getEventSchemasTool', () => {
  it('returns a known event schema', async () => {
    const out = await getEventSchemasTool.execute(
      { event_names: ['app/user.created'] },
      ctx,
    );
    expect(out.observation).toContain('"id": "string"');
  });

  it('notes when a schema is unavailable', async () => {
    const out = await getEventSchemasTool.execute(
      { event_names: ['app/user.deleted'] },
      ctx,
    );
    expect(out.observation).toContain('no schema available');
  });

  it('rejects invalid input with an observation, not a throw', async () => {
    const out = await getEventSchemasTool.execute({ event_names: [] }, ctx);
    expect(out.observation).toContain('Invalid input');
  });
});

describe('submitQueryTool', () => {
  it('records the draft patch and a step.completed publish', async () => {
    const out = await submitQueryTool.execute(
      {
        sql: "SELECT count() FROM runs WHERE status = 'Failed'",
        title: 'Failed runs',
        reasoning: 'counts failed runs',
        event_names: [],
      },
      ctx,
    );

    expect(out.draftPatch?.sql).toContain('FROM runs');
    expect(out.draftPatch?.selectedEvents).toEqual([]);
    expect(out.publish?.event).toBe('step.completed');
    expect(out.publish?.data.step).toBe('query-writer');
    expect(out.observation).toContain('summary');
  });

  it('maps event_names into selectedEvents', async () => {
    const out = await submitQueryTool.execute(
      {
        sql: 'SELECT * FROM events',
        title: 'T',
        reasoning: 'r',
        event_names: ['app/user.created'],
      },
      ctx,
    );
    expect(out.draftPatch?.selectedEvents).toEqual([
      { event_name: 'app/user.created', reason: 'used in query' },
    ]);
  });

  it('rejects a missing sql with an observation', async () => {
    const out = await submitQueryTool.execute({ title: 'T' }, ctx);
    expect(out.observation).toContain('Invalid input');
    expect(out.draftPatch).toBeUndefined();
  });
});
