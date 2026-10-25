import { ReplayStatus } from '@inngest/components/types/replay';
import { describe, expect, it } from 'vitest';

import { getReplayStatus } from './getReplayStatus';

describe('getReplayStatus', () => {
  it.each([
    { endedAt: null, error: null, expected: ReplayStatus.Created },
    {
      endedAt: null,
      error: 'Retrying',
      expected: ReplayStatus.Created,
    },
    {
      endedAt: '2026-10-15T12:00:00Z',
      error: null,
      expected: ReplayStatus.Ended,
    },
    {
      endedAt: '2026-10-15T12:00:00Z',
      error: 'Retries exhausted',
      expected: ReplayStatus.Failed,
    },
    {
      endedAt: '2026-10-15T12:00:00Z',
      error: 'Invalid payload',
      expected: ReplayStatus.Failed,
    },
    {
      endedAt: '2026-10-15T12:00:00Z',
      error: '',
      expected: ReplayStatus.Failed,
    },
  ])('maps endedAt=$endedAt and error=$error to $expected', (replay) => {
    expect(getReplayStatus(replay)).toBe(replay.expected);
  });
});
