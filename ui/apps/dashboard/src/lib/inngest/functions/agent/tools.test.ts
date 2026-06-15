import { describe, it, expect } from 'vitest';

import {
  listDataSourcesTool,
  describeTableTool,
  findEventsTool,
  getEventSchemasTool,
  submitQueryTool,
} from './tools';

const ctx = {
  clientState: {
    eventTypes: [
      'app/user.created',
      'app/user.deleted',
      'billing/charge.succeeded',
    ],
    schemas: [{ name: 'app/user.created', schema: '{ "id": "string" }' }],
    currentQuery: 'SELECT 1',
  },
};

describe('listDataSourcesTool', () => {
  it('lists events, runs, and steps', async () => {
    const out = await listDataSourcesTool.execute({}, ctx as any);
    expect(out.observation).toContain('runs:');
    expect(out.observation).toContain('steps:');
  });
});

describe('describeTableTool', () => {
  it('describes the runs table columns', async () => {
    const out = await describeTableTool.execute(
      { table_name: 'runs' },
      ctx as any,
    );
    expect(out.observation).toContain('status');
  });
});

describe('findEventsTool', () => {
  it('lists all events with no search term', async () => {
    const out = await findEventsTool.execute({}, ctx as any);
    expect(out.observation).toContain('app/user.created');
    expect(out.observation).toContain('billing/charge.succeeded');
  });
  it('filters events case-insensitively', async () => {
    const out = await findEventsTool.execute({ search: 'USER' }, ctx as any);
    expect(out.observation).toContain('app/user.created');
    expect(out.observation).not.toContain('billing/charge.succeeded');
  });
});

describe('getEventSchemasTool', () => {
  it('returns a known event schema', async () => {
    const out = await getEventSchemasTool.execute(
      { event_names: ['app/user.created'] },
      ctx as any,
    );
    expect(out.observation).toContain('"id": "string"');
  });
  it('notes when a schema is unavailable', async () => {
    const out = await getEventSchemasTool.execute(
      { event_names: ['app/user.deleted'] },
      ctx as any,
    );
    expect(out.observation).toMatch(/no schema|not available/i);
  });
});

describe('submitQueryTool', () => {
  it('records draftPatch (incl. tables) and a step.completed publish', async () => {
    const out = await submitQueryTool.execute(
      {
        sql: "SELECT count() FROM runs WHERE status = 'Failed'",
        title: 'Failed runs',
        reasoning: 'counts failed runs',
        tables: ['runs'],
        event_names: [],
      },
      ctx as any,
    );
    expect(out.draftPatch?.sql).toContain('FROM runs');
    expect(out.draftPatch?.tables).toEqual(['runs']);
    expect(out.draftPatch?.selectedEvents).toEqual([]);
    expect(out.publish?.event).toBe('step.completed');
    expect(out.publish?.data.step).toBe('query-writer');
    expect(out.publish?.data.tables).toEqual(['runs']);
  });

  it('maps event_names into selectedEvents when querying events', async () => {
    const out = await submitQueryTool.execute(
      {
        sql: 'SELECT * FROM events',
        title: 'T',
        reasoning: 'r',
        tables: ['events'],
        event_names: ['app/user.created'],
      },
      ctx as any,
    );
    expect(out.draftPatch?.selectedEvents).toEqual([
      { event_name: 'app/user.created', reason: 'used in query' },
    ]);
  });
});
