import { describe, expect, it } from 'vitest';

import { SandboxStatus } from '@/gql/graphql';

import { formatDuration, formatTimeout, isActiveSandboxStatus } from './utils';

describe('formatDuration', () => {
  it.each([
    [450, '450ms'],
    [12_300, '12.3s'],
    [37 * 60_000, '37m'],
    [59.6 * 60_000, '1h'],
    [60 * 60_000, '1h'],
    [125 * 60_000, '2h 5m'],
    [24 * 60 * 60_000, '1d'],
    [(48 + 3) * 60 * 60_000, '2d 3h'],
  ])('formats %d ms as %s', (milliseconds, expected) => {
    expect(formatDuration(milliseconds)).toBe(expected);
  });
});

describe('formatTimeout', () => {
  it.each([
    [0, 'No timeout'],
    [900, '15m'],
    [6 * 60 * 60, '6h'],
  ])('formats %d seconds as %s', (seconds, expected) => {
    expect(formatTimeout(seconds)).toBe(expected);
  });
});

describe('isActiveSandboxStatus', () => {
  it.each([
    [SandboxStatus.Pending, true],
    [SandboxStatus.Running, true],
    [SandboxStatus.Stopping, true],
    [SandboxStatus.Stopped, false],
    [SandboxStatus.Failed, false],
    [SandboxStatus.LaunchUnknown, false],
  ])('treats %s as active: %s', (status, expected) => {
    expect(isActiveSandboxStatus(status)).toBe(expected);
  });
});
