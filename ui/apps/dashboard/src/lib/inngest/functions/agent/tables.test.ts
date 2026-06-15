import { describe, it, expect } from 'vitest';

import { TABLES, getTable, describeTable, listDataSources } from './tables';

describe('table registry', () => {
  it('includes events, runs, and steps', () => {
    const names = TABLES.map((t) => t.name);
    expect(names).toEqual(expect.arrayContaining(['events', 'runs', 'steps']));
  });

  it('marks only events as dynamic', () => {
    expect(getTable('events')?.dynamic).toBe(true);
    expect(getTable('runs')?.dynamic).toBeFalsy();
  });

  it('runs table exposes status and triggering_event_name columns', () => {
    const cols = getTable('runs')!.columns.map((c) => c.name);
    expect(cols).toEqual(
      expect.arrayContaining(['status', 'triggering_event_name']),
    );
  });

  it('listDataSources lists every table with its purpose', () => {
    const out = listDataSources();
    expect(out).toContain('events:');
    expect(out).toContain('runs:');
    expect(out).toContain('steps:');
  });

  it('describeTable returns columns for a known table and a hint for unknown', () => {
    expect(describeTable('runs')).toContain('status');
    expect(describeTable('nope')).toMatch(/unknown table/i);
  });

  it('describeTable points events at the event tools', () => {
    expect(describeTable('events')).toMatch(/find_events|get_event_schemas/);
  });
});
