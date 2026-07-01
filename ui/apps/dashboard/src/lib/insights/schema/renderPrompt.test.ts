import { describe, expect, it } from 'vitest';

import { insightsSchemaCatalog } from './catalog';
import { renderInsightsSchemaForPrompt } from './renderPrompt';
import type { InsightsSchemaCatalog } from './types';

describe('renderInsightsSchemaForPrompt', () => {
  it('renders the compact prompt XML shape', () => {
    const catalog: InsightsSchemaCatalog = {
      version: 'test',
      tables: [
        {
          name: 'runs',
          description: 'Function executions.',
          notes: ['Rows: One row per run', 'Use for: run status'],
          defaultTimeColumn: 'queued_at',
          columns: [
            {
              name: 'id',
              type: 'ULID String',
              description: 'Unique identifier for the run.',
            },
            {
              name: 'function_id',
              type: 'UUID',
              description: 'Function slug.',
              notes: ['Write complete function slug strings with = or IN'],
              examples: [
                "function_id = 'app-fn'",
                "function_id IN ('app-a', 'app-b')",
              ],
            },
            {
              name: 'inngest',
              type: 'Map(String, Dynamic)',
              description: 'System metadata.',
              children: [
                {
                  name: 'ai.values.model',
                  type: 'String',
                  description: 'Model name.',
                },
              ],
            },
          ],
        },
      ],
    };

    const output = renderInsightsSchemaForPrompt(catalog);

    expect(output).toContain('<insights_tables version="test">');
    expect(output).toContain(
      '<table name="runs" default_time_column="queued_at" description="Function executions.">',
    );
    expect(output).toContain(
      '<notes>Rows: One row per run. Use for: run status.</notes>',
    );
    expect(output).toContain(
      '<column name="id" type="ULID String" description="Unique identifier for the run." />',
    );
    expect(output).toContain(
      '<column name="function_id" type="UUID" description="Function slug.">',
    );
    expect(output).toContain(
      '<notes>Write complete function slug strings with = or IN.</notes>',
    );
    expect(output).toContain(
      "<examples>function_id = 'app-fn'; function_id IN ('app-a', 'app-b')</examples>",
    );
    expect(output).toContain('<children>');
    expect(output).toContain(
      '<column name="ai.values.model" type="String" description="Model name." />',
    );
    expect(output).not.toContain('<description>');
    expect(output).not.toContain('<guidance>');
    expect(output).not.toContain('<example>');
  });

  it('includes critical fields from the shared Insights catalog', () => {
    const output = renderInsightsSchemaForPrompt(insightsSchemaCatalog);

    expect(output).toContain('<insights_tables version="2026-06-29">');
    expect(output).toContain(
      '<table name="step_attempts" default_time_column="queued_at" description="Every step attempt, including retries.">',
    );
    expect(output).toContain(
      'Use for: retry analysis; true AI token totals; step-level failures including retried attempts.',
    );
    expect(output).toContain(
      '<table name="metadata" default_time_column="updated_at" description="Span-level system and user metadata.">',
    );
    expect(output).toContain(
      '<column name="inngest" type="Map(String, Tuple(updated_at DateTime, values Dynamic))" description="System-defined metadata emitted by Inngest.">',
    );
    expect(output).toContain(
      '<column name="&lt;kind&gt;.values" type="Dynamic" description="System-defined metadata payload for an arbitrary Inngest kind." />',
    );
  });

  it('escapes XML-sensitive text', () => {
    const catalog: InsightsSchemaCatalog = {
      version: 'test"&<>',
      tables: [
        {
          name: 'events',
          description: 'Use <data> & "names"',
          notes: ['Rows: One > row.', 'Use for: a < b & c > d.'],
          columns: [
            {
              name: 'data',
              type: 'JSONString',
              description: 'A & B < C',
              examples: ['data.a < data.b', 'data.c & data.d'],
            },
          ],
        },
      ],
    };

    const output = renderInsightsSchemaForPrompt(catalog);

    expect(output).toContain(
      '<insights_tables version="test&quot;&amp;&lt;&gt;">',
    );
    expect(output).toContain(
      '<table name="events" description="Use &lt;data&gt; &amp; &quot;names&quot;">',
    );
    expect(output).toContain('a &lt; b &amp; c &gt; d');
    expect(output).toContain(
      '<column name="data" type="JSONString" description="A &amp; B &lt; C">',
    );
    expect(output).toContain(
      '<examples>data.a &lt; data.b; data.c &amp; data.d</examples>',
    );
  });
});
