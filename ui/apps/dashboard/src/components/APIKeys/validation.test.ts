import { describe, expect, it } from 'vitest';

import {
  API_KEY_NAME_MAX,
  validateAPIKeyName,
  validateRestrictedAPIKey,
} from '@/components/APIKeys/validation';

describe('validateAPIKeyName', () => {
  it('rejects an empty string', () => {
    expect(validateAPIKeyName('')).toBe('Name is required.');
  });

  it('rejects whitespace-only input', () => {
    expect(validateAPIKeyName('   ')).toBe('Name is required.');
    expect(validateAPIKeyName('\t\n')).toBe('Name is required.');
  });

  it('accepts a trimmed name with surrounding whitespace', () => {
    expect(validateAPIKeyName('  my-key  ')).toBeNull();
  });

  it('accepts a name exactly at the max length', () => {
    const name = 'a'.repeat(API_KEY_NAME_MAX);
    expect(validateAPIKeyName(name)).toBeNull();
  });

  it('rejects a name one character over the max length', () => {
    const name = 'a'.repeat(API_KEY_NAME_MAX + 1);
    expect(validateAPIKeyName(name)).toBe(
      `Name must be ${API_KEY_NAME_MAX} characters or fewer.`,
    );
  });

  it('measures length on the trimmed value, not the raw input', () => {
    // Padded with whitespace but content fits within the limit.
    const padded = '   ' + 'a'.repeat(API_KEY_NAME_MAX) + '   ';
    expect(validateAPIKeyName(padded)).toBeNull();
  });
});

describe('restricted keys', () => {
  const input = {
    name: 'nightly-sync',
    allEnvironments: false,
    workspaceID: undefined,
    permissions: ['apps:read:*'],
  };

  it('requires an explicit environment by default', () => {
    expect(validateRestrictedAPIKey(input).environment).toBe(
      'Select an environment.',
    );
    expect(
      validateRestrictedAPIKey({ ...input, workspaceID: 'production-id' }),
    ).toEqual({ name: null, environment: null, permissions: null });
  });

  it('allows an explicit all-environment choice', () => {
    expect(
      validateRestrictedAPIKey({ ...input, allEnvironments: true }),
    ).toEqual({ name: null, environment: null, permissions: null });
  });

  it('requires permissions and a name', () => {
    expect(
      validateRestrictedAPIKey({
        ...input,
        allEnvironments: true,
        permissions: [],
      }).permissions,
    ).toBe('Select at least one permission.');
    expect(
      validateRestrictedAPIKey({ ...input, name: ' ', allEnvironments: true })
        .name,
    ).toBe('Name is required.');
  });

  it('reports all missing fields together', () => {
    expect(
      validateRestrictedAPIKey({ ...input, name: '', permissions: [] }),
    ).toEqual({
      name: 'Name is required.',
      environment: 'Select an environment.',
      permissions: 'Select at least one permission.',
    });
  });
});
