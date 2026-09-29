import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  ACCOUNT_CREATED_EVENT,
  normalizeEmail,
  sha256Hex,
  trackAccountCreated,
} from './accountCreatedTracking';

type Pushed = Record<string, unknown> & { eventCallback?: () => void };

let dataLayer: Pushed[];
let storage: Map<string, string>;

const stubWindow = ({ gtm = true } = {}) => {
  dataLayer = [];
  vi.stubGlobal('window', {
    dataLayer,
    ...(gtm && { google_tag_manager: {} }),
    localStorage: {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
    },
  });
};

beforeEach(() => {
  storage = new Map();
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe('sha256Hex', () => {
  it('normalizes emails before hashing', () => {
    expect(normalizeEmail('  Pat@Example.COM ')).toBe('pat@example.com');
  });

  it('returns a lowercase hex SHA-256 digest', async () => {
    expect(await sha256Hex('abc')).toBe(
      'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad',
    );
  });
});

describe('trackAccountCreated', () => {
  it('pushes the account ID and a hashed email, then waits for GTM', async () => {
    stubWindow();
    const done = trackAccountCreated({
      accountID: 'acct_1',
      email: ' Pat@Example.com ',
      isFirstOrganization: true,
    });
    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));

    const pushed = dataLayer[0];
    expect(pushed).toMatchObject({
      event: ACCOUNT_CREATED_EVENT,
      account_id: 'acct_1',
      user_data: {
        sha256_email_address: await sha256Hex('pat@example.com'),
      },
      eventTimeout: 2000,
    });
    expect(JSON.stringify(pushed)).not.toContain('Example.com');

    pushed.eventCallback?.();
    await expect(done).resolves.toBeUndefined();
  });

  it('stops waiting after the timeout if GTM never calls back', async () => {
    vi.useFakeTimers();
    stubWindow();
    const done = trackAccountCreated({
      accountID: 'acct_1',
      email: 'pat@example.com',
      isFirstOrganization: true,
      timeoutMs: 500,
    });
    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));
    await vi.advanceTimersByTimeAsync(500);
    await expect(done).resolves.toBeUndefined();
  });

  it('does not wait when GTM is not on the page', async () => {
    stubWindow({ gtm: false });
    await trackAccountCreated({
      accountID: 'acct_1',
      email: undefined,
      isFirstOrganization: true,
    });
    expect(dataLayer).toHaveLength(1);
    expect(dataLayer[0]).not.toHaveProperty('user_data');
  });

  it('fires once per account', async () => {
    stubWindow({ gtm: false });
    const input = {
      accountID: 'acct_1',
      email: 'pat@example.com',
      isFirstOrganization: true,
    };
    await trackAccountCreated(input);
    await trackAccountCreated(input);
    expect(dataLayer).toHaveLength(1);
  });

  it.each([
    { name: 'additional organizations', accountID: 'acct_1', first: false },
    { name: 'a missing account ID', accountID: null, first: true },
  ])('skips $name', async ({ accountID, first }) => {
    stubWindow({ gtm: false });
    await trackAccountCreated({
      accountID,
      email: 'pat@example.com',
      isFirstOrganization: first,
    });
    expect(dataLayer).toHaveLength(0);
  });

  it('does nothing during server rendering', async () => {
    await expect(
      trackAccountCreated({
        accountID: 'acct_1',
        email: 'pat@example.com',
        isFirstOrganization: true,
      }),
    ).resolves.toBeUndefined();
  });
});
