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

const stubWindow = ({ gtm = true, autoCallback = false } = {}) => {
  dataLayer = [];
  if (autoCallback) {
    // Behave like GTM: run the event's tags, then call eventCallback.
    dataLayer.push = (item: Pushed) => {
      Array.prototype.push.call(dataLayer, item);
      item.eventCallback?.();
      return dataLayer.length;
    };
  }
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

describe('normalizeEmail gmail handling', () => {
  it.each([
    ['John.Doe@Gmail.com', 'johndoe@gmail.com'],
    [' j.o.h.n@googlemail.com ', 'john@googlemail.com'],
    ['pat.holcomb@inngest.com', 'pat.holcomb@inngest.com'],
    ['first.last@notgmail.com', 'first.last@notgmail.com'],
    ['not.an.email', 'not.an.email'],
  ])('normalizes %s', (input, expected) => {
    expect(normalizeEmail(input)).toBe(expected);
  });
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

  it('gives up after a short wait when GTM never loads', async () => {
    vi.useFakeTimers();
    stubWindow({ gtm: false });
    const done = trackAccountCreated({
      accountID: 'acct_1',
      email: undefined,
      isFirstOrganization: true,
    });
    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));
    expect(dataLayer[0]).not.toHaveProperty('user_data');
    await vi.advanceTimersByTimeAsync(1000);
    await expect(done).resolves.toBeUndefined();
  });

  it('waits for GTM that loads after the push', async () => {
    vi.useFakeTimers();
    stubWindow({ gtm: false });
    let resolved = false;
    const done = trackAccountCreated({
      accountID: 'acct_1',
      email: 'pat@example.com',
      isFirstOrganization: true,
    }).then(() => {
      resolved = true;
    });
    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));

    await vi.advanceTimersByTimeAsync(300);
    expect(resolved).toBe(false);

    (window as Window & { google_tag_manager?: unknown }).google_tag_manager =
      {};
    await vi.advanceTimersByTimeAsync(100);
    expect(resolved).toBe(false);

    dataLayer[0].eventCallback?.();
    await done;
    expect(storage.size).toBe(1);
  });

  it('allows a retry when GTM never loaded', async () => {
    vi.useFakeTimers();
    stubWindow({ gtm: false });
    const input = {
      accountID: 'acct_1',
      email: undefined,
      isFirstOrganization: true,
    };

    const first = trackAccountCreated(input);
    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));
    await vi.advanceTimersByTimeAsync(1000);
    await first;
    expect(storage.size).toBe(0);

    const second = trackAccountCreated(input);
    await vi.waitFor(() => expect(dataLayer).toHaveLength(2));
    await vi.advanceTimersByTimeAsync(1000);
    await second;
  });

  it('fires once per account', async () => {
    stubWindow({ autoCallback: true });
    const input = {
      accountID: 'acct_1',
      email: 'pat@example.com',
      isFirstOrganization: true,
    };
    await trackAccountCreated(input);
    await trackAccountCreated(input);
    expect(dataLayer).toHaveLength(1);
  });

  it('ignores a concurrent call for the same account', async () => {
    stubWindow();
    const input = {
      accountID: 'acct_1',
      email: 'pat@example.com',
      isFirstOrganization: true,
    };
    const first = trackAccountCreated(input);
    await trackAccountCreated(input);

    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));
    dataLayer[0].eventCallback?.();
    await first;
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
