import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  normalizeEmail,
  sha256Hex,
  trackAccountCreated,
} from './accountCreatedTracking';

type Pushed = Record<string, unknown> & { eventCallback?: () => void };
let dataLayer: Pushed[];
const claimConversion = vi.fn<(signal: AbortSignal) => Promise<boolean>>();
const input = { accountID: 'acct_1', email: undefined, claimConversion };

const stubWindow = (gtm = true) => {
  dataLayer = [];
  vi.stubGlobal('window', {
    dataLayer,
    ...(gtm && { google_tag_manager: {} }),
  });
};

beforeEach(() => {
  claimConversion.mockReset().mockResolvedValue(true);
  stubWindow();
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe('normalizeEmail', () => {
  it.each([
    ['John.Doe@Gmail.com', 'johndoe@gmail.com'],
    [' j.o.h.n+Shopping@googlemail.com ', 'john@googlemail.com'],
    ['Jane.Doe+Shopping@gmail.com', 'janedoe@gmail.com'],
    ['pat.holcomb+test@inngest.com', 'pat.holcomb+test@inngest.com'],
    ['first.last@notgmail.com', 'first.last@notgmail.com'],
    ['not.an.email', 'not.an.email'],
    ['  Pat@Example.COM ', 'pat@example.com'],
  ])('normalizes %s', (email, expected) => {
    expect(normalizeEmail(email)).toBe(expected);
  });
});

it('returns a lowercase SHA-256 digest', async () => {
  expect(await sha256Hex('abc')).toBe(
    'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad',
  );
});

describe('trackAccountCreated', () => {
  it('claims the conversion, pushes a hashed email, and waits for GTM', async () => {
    const done = trackAccountCreated({ ...input, email: ' Pat@Example.com ' });
    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));
    expect(claimConversion).toHaveBeenCalledOnce();
    expect(dataLayer[0]).toMatchObject({
      event: 'account_created',
      account_id: 'acct_1',
      user_data: { sha256_email_address: await sha256Hex('pat@example.com') },
      eventTimeout: 2000,
    });
    expect(JSON.stringify(dataLayer[0])).not.toContain('Example.com');
    dataLayer[0].eventCallback?.();
    await expect(done).resolves.toBeUndefined();
  });

  it('stops waiting if GTM never calls back', async () => {
    vi.useFakeTimers();
    const done = trackAccountCreated(input);
    await vi.advanceTimersByTimeAsync(2000);
    await done;
    expect(dataLayer).toHaveLength(1);
  });

  it('does not consume a claim when GTM is blocked', async () => {
    vi.useFakeTimers();
    stubWindow(false);
    const done = trackAccountCreated(input);
    await vi.advanceTimersByTimeAsync(1000);
    await done;
    expect(claimConversion).not.toHaveBeenCalled();
    expect(dataLayer).toHaveLength(0);
  });

  it('waits for GTM before claiming', async () => {
    vi.useFakeTimers();
    stubWindow(false);
    const done = trackAccountCreated(input);
    await vi.advanceTimersByTimeAsync(300);
    expect(claimConversion).not.toHaveBeenCalled();
    Object.assign(window, { google_tag_manager: {} });
    await vi.advanceTimersByTimeAsync(50);
    expect(dataLayer).toHaveLength(1);
    dataLayer[0].eventCallback?.();
    await done;
  });

  it('allows another attempt after GTM was blocked', async () => {
    vi.useFakeTimers();
    stubWindow(false);
    const first = trackAccountCreated(input);
    await vi.advanceTimersByTimeAsync(1000);
    await first;
    stubWindow();
    const second = trackAccountCreated(input);
    await vi.advanceTimersByTimeAsync(0);
    expect(claimConversion).toHaveBeenCalledOnce();
    expect(dataLayer).toHaveLength(1);
    dataLayer[0].eventCallback?.();
    await second;
  });

  it('skips conversions already claimed by another browser or an earlier visit', async () => {
    claimConversion.mockResolvedValue(false);
    await trackAccountCreated(input);
    expect(dataLayer).toHaveLength(0);
  });

  it('ignores overlapping calls', async () => {
    const first = trackAccountCreated(input);
    await trackAccountCreated(input);
    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));
    expect(claimConversion).toHaveBeenCalledOnce();
    dataLayer[0].eventCallback?.();
    await first;
  });

  it('does not block signup when claiming fails', async () => {
    claimConversion.mockRejectedValue(new Error('unavailable'));
    await expect(trackAccountCreated(input)).resolves.toBeUndefined();
    expect(dataLayer).toHaveLength(0);
  });

  it('aborts a slow claim and ignores its late result', async () => {
    vi.useFakeTimers();
    let complete: (claimed: boolean) => void = () => {};
    claimConversion.mockImplementation(
      () =>
        new Promise((resolve) => {
          complete = resolve;
        }),
    );
    const done = trackAccountCreated(input);
    await vi.advanceTimersByTimeAsync(2000);
    await done;
    expect(claimConversion.mock.calls[0][0].aborted).toBe(true);
    complete(true);
    await vi.advanceTimersByTimeAsync(0);
    expect(dataLayer).toHaveLength(0);
  });

  it('bounds hashing without consuming the claim', async () => {
    vi.useFakeTimers();
    vi.stubGlobal('crypto', {
      subtle: { digest: () => new Promise(() => {}) },
    });
    const done = trackAccountCreated({ ...input, email: 'pat@example.com' });
    await vi.advanceTimersByTimeAsync(2000);
    await done;
    expect(claimConversion).not.toHaveBeenCalled();
  });

  it('still tracks if hashing fails', async () => {
    vi.stubGlobal('crypto', {
      subtle: {
        digest: async () => {
          throw new Error('unavailable');
        },
      },
    });
    const done = trackAccountCreated({ ...input, email: 'pat@example.com' });
    await vi.waitFor(() => expect(dataLayer).toHaveLength(1));
    expect(dataLayer[0]).not.toHaveProperty('user_data');
    dataLayer[0].eventCallback?.();
    await done;
  });

  it('skips a missing account', async () => {
    await trackAccountCreated({ ...input, accountID: null });
    expect(claimConversion).not.toHaveBeenCalled();
  });

  it('does nothing during server rendering', async () => {
    vi.stubGlobal('window', undefined);
    await trackAccountCreated(input);
    expect(claimConversion).not.toHaveBeenCalled();
  });
});
