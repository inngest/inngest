import { describe, expect, it } from 'vitest';

import { buildSystemPrompt } from './index';

describe('buildSystemPrompt', () => {
  it('injects the generated Insights schema into the query writer prompt', () => {
    const prompt = buildSystemPrompt({
      selectedEvents: [],
      schemas: [],
      query: 'Show token usage by model for the last 7 days',
    });

    expect(prompt).toContain('<insights_tables version="2026-06-29">');
    expect(prompt).toContain(
      'Forbidden columns: `account_id` and `workspace_id` are injected automatically',
    );
    expect(prompt).toContain('<table name="step_attempts"');
  });

  it('keeps selected event schemas separate from the global Insights schema', () => {
    const prompt = buildSystemPrompt({
      selectedEvents: [
        {
          event_name: 'app/user.created',
          reason: 'The user asked about user creation.',
        },
      ],
      schemas: [
        {
          name: 'app/user.created',
          schema: JSON.stringify({
            type: 'object',
            properties: {
              data: {
                type: 'object',
                properties: {
                  email: { type: 'string' },
                },
              },
            },
          }),
        },
        {
          name: 'app/ignored',
          schema: JSON.stringify({ type: 'object' }),
        },
      ],
      query: 'Show user creations by email domain',
    });

    expect(prompt).toContain('<event name="app&#x2F;user.created">');
    expect(prompt).toContain('"email":{"type":"string"}');
    expect(prompt).not.toContain('<event name="app/ignored">');
  });
});
