import { describe, expect, it } from 'vitest';

import { getCronTriggerMetadata } from './cronTrigger';

describe('getCronTriggerMetadata', () => {
  it('extracts scheduledAt and fireAt from a cron payload', () => {
    const metadata = getCronTriggerMetadata([
      JSON.stringify({
        data: {
          cron: '0 * * * *',
          scheduledAt: '2026-04-07T10:00:00Z',
          fireAt: '2026-04-07T10:04:12Z',
        },
      }),
    ]);

    expect(metadata.scheduledAt?.toISOString()).toBe('2026-04-07T10:00:00.000Z');
    expect(metadata.fireAt?.toISOString()).toBe('2026-04-07T10:04:12.000Z');
  });

  it('returns nulls when payload is absent or malformed', () => {
    expect(getCronTriggerMetadata([])).toEqual({ scheduledAt: null, fireAt: null });
    expect(getCronTriggerMetadata(['not-json'])).toEqual({ scheduledAt: null, fireAt: null });
  });
});
