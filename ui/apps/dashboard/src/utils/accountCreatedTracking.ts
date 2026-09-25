// Browser-side `account_created` event for the ad-platform conversion tags in
// GTM (Google Ads, LinkedIn, X, Reddit, OpenAI). The server-side Segment
// "Account Created" event remains the source of truth for analytics; this one
// exists because those tags only run in the browser.

export const ACCOUNT_CREATED_EVENT = 'account_created';
export const ACCOUNT_CREATED_TIMEOUT_MS = 2000;

const TRACKED_KEY_PREFIX = 'inngest_account_created_tracked:';

type GTMWindow = Window & {
  dataLayer?: unknown[];
  google_tag_manager?: unknown;
};

export const normalizeEmail = (email: string): string =>
  email.trim().toLowerCase();

export const sha256Hex = async (value: string): Promise<string | undefined> => {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle) return undefined;

  const digest = await subtle.digest(
    'SHA-256',
    new TextEncoder().encode(value),
  );
  return Array.from(new Uint8Array(digest), (byte) =>
    byte.toString(16).padStart(2, '0'),
  ).join('');
};

const hasTracked = (accountID: string): boolean => {
  try {
    return window.localStorage.getItem(TRACKED_KEY_PREFIX + accountID) !== null;
  } catch {
    return false;
  }
};

const markTracked = (accountID: string): void => {
  try {
    window.localStorage.setItem(TRACKED_KEY_PREFIX + accountID, '1');
  } catch {
    // Storage can be unavailable (private mode, blocked site data); tracking
    // then relies on Google Ads' transaction ID dedupe alone.
  }
};

type TrackAccountCreatedInput = {
  accountID: string | null | undefined;
  email: string | null | undefined;
  /** Only a user's first organization counts as a new signup for ads. */
  isFirstOrganization: boolean;
  timeoutMs?: number;
};

/**
 * Pushes `account_created` to the GTM dataLayer and resolves once GTM reports
 * that the tags it triggered have run, or after `timeoutMs`, so the caller can
 * navigate away without cutting those requests off. Only a SHA-256 hash of the
 * normalized email is pushed, never the raw address. Never rejects.
 */
export async function trackAccountCreated({
  accountID,
  email,
  isFirstOrganization,
  timeoutMs = ACCOUNT_CREATED_TIMEOUT_MS,
}: TrackAccountCreatedInput): Promise<void> {
  if (typeof window === 'undefined' || !accountID || !isFirstOrganization) {
    return;
  }
  // /organization-setup returns the existing account on reload or revisit;
  // count each account once.
  if (hasTracked(accountID)) return;
  markTracked(accountID);

  let hashedEmail: string | undefined;
  try {
    hashedEmail = email ? await sha256Hex(normalizeEmail(email)) : undefined;
  } catch {
    hashedEmail = undefined;
  }

  const w = window as GTMWindow;
  w.dataLayer = w.dataLayer || [];
  const dataLayer = w.dataLayer;
  // Without GTM on the page (e.g. Segment blocked), nothing would ever call
  // eventCallback, so don't wait.
  const waitMs = w.google_tag_manager ? timeoutMs : 0;

  await new Promise<void>((resolve) => {
    const timer = setTimeout(resolve, waitMs);
    dataLayer.push({
      event: ACCOUNT_CREATED_EVENT,
      account_id: accountID,
      ...(hashedEmail && { user_data: { sha256_email_address: hashedEmail } }),
      eventCallback: () => {
        clearTimeout(timer);
        resolve();
      },
      eventTimeout: timeoutMs,
    });
  });
}
