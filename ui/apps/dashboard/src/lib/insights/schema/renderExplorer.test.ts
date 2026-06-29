import { describe, expect, it } from 'vitest';

import { insightsSchemaCatalog } from './catalog';
import { renderInsightsSchemaForExplorer } from './renderExplorer';

describe('renderInsightsSchemaForExplorer', () => {
  it('renders table entries from the shared catalog', () => {
    const { entries } = renderInsightsSchemaForExplorer(insightsSchemaCatalog);

    expect(entries.map((entry) => entry.key)).toEqual([
      'events',
      'runs',
      'steps',
      'step_attempts',
      'extended_trace_spans',
      'metadata',
    ]);
  });

  it('adds metadata for tables and columns by schema path', () => {
    const { metadataByPath } = renderInsightsSchemaForExplorer(
      insightsSchemaCatalog,
    );

    expect(metadataByPath['step_attempts']?.description).toBe(
      'Every step attempt, including retries.',
    );
    expect(metadataByPath['step_attempts']?.notes).toContain(
      'Rows: One row per step attempt.',
    );
    expect(metadataByPath['step_attempts.function_id']?.notes).toContain(
      'Write complete function slug strings with = or IN.',
    );
    expect(metadataByPath['metadata.level']?.notes).toContain(
      'This is not a log level.',
    );
    expect(metadataByPath['metadata.inngest']?.notes).toContain(
      'Use dot syntax for fixed metadata paths.',
    );
  });

  it('renders nested metadata children', () => {
    const { entries, metadataByPath } = renderInsightsSchemaForExplorer(
      insightsSchemaCatalog,
    );
    const steps = entries.find((entry) => entry.key === 'steps')?.node;

    expect(steps?.kind).toBe('table');
    if (steps?.kind !== 'table') throw new Error('steps table not rendered');

    const inngest = steps.children.find(
      (child) => child.path === 'steps.inngest',
    );
    expect(inngest?.kind).toBe('object');
    if (inngest?.kind !== 'object')
      throw new Error('inngest node not rendered');

    expect(
      inngest.children.some(
        (child) => child.path === 'steps.inngest.<kind>.values',
      ),
    ).toBe(true);
    expect(metadataByPath['steps.inngest.<kind>.values']?.description).toBe(
      'System-defined metadata payload for an arbitrary Inngest kind.',
    );
  });
});
