// Browser-side `account_created` event for the ad-platform conversion tags in
// GTM (Google Ads, LinkedIn, X, Reddit, OpenAI). The server-side Segment
// "Account Created" event remains the source of truth for analytics; this one
// exists because those tags only run in the browser.

export const ACCOUNT_CREATED_EVENT = 'account_created';
export const ACCOUNT_CREATED_TIMEOUT_MS = 2000;

const TRACKED_KEY_PREFIX = 'inngest_account_created_tracked:';

// GTM loads asynchronously. If it is not on the page yet, the push below is
// queued in the dataLayer and GTM runs it once it initializes, but only if the
// page stays alive long enough. Give GTM a short window to appear before
// concluding it is blocked (ad blockers, consent tools).
const GTM_LOAD_WAIT_MS = 1000;
const GTM_POLL_MS = 50;

// Calls for the same account that overlap while the first is still hashing or
// waiting (React strict mode, a double-fired callback) should push only once.
const inFlight = new Set<string>();

type GTMWindow = Window & {
  dataLayer?: unknown[];
  google_tag_manager?: unknown;
};

/**
 * Normalizes an email the way Google's enhanced conversions expects before
 * hashing: trim, lowercase, and for gmail.com / googlemail.com strip dots from
 * the username (john.doe@gmail.com and johndoe@gmail.com are the same inbox).
 */
export const normalizeEmail = (email: string): string => {
  const normalized = email.trim().toLowerCase();
  const at = normalized.lastIndexOf('@');
  if (at < 0) return normalized;
  const domain = normalized.slice(at + 1);
  if (domain !== 'gmail.com' && domain !== 'googlemail.com') return normalized;
  return `${normalized.slice(0, at).replace(/\./g, '')}@${domain}`;
};

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
 * navigate away without cutting those requests off. If GTM has not loaded yet
 * it waits briefly for it to appear; if it never does, it gives up and does not
 * mark the account as tracked, so a later visit can try again. Only a SHA-256
 * hash of the normalized email is pushed, never the raw address. Never rejects.
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
  if (hasTracked(accountID) || inFlight.has(accountID)) return;
  inFlight.add(accountID);

  try {
    let hashedEmail: string | undefined;
    try {
      hashedEmail = email ? await sha256Hex(normalizeEmail(email)) : undefined;
    } catch {
      hashedEmail = undefined;
    }

    const w = window as GTMWindow;
    w.dataLayer = w.dataLayer || [];
    const dataLayer = w.dataLayer;

    await new Promise<void>((resolve) => {
      let pollTimer: ReturnType<typeof setInterval> | undefined;
      const finish = () => {
        clearTimeout(timeoutTimer);
        clearInterval(pollTimer);
        resolve();
      };
      const timeoutTimer = setTimeout(finish, timeoutMs);

      if (!w.google_tag_manager) {
        // Stop early only if GTM never shows up. Once it does, the remaining
        // wait is for eventCallback or the overall timeout.
        let waited = 0;
        pollTimer = setInterval(() => {
          if (w.google_tag_manager) {
            clearInterval(pollTimer);
            return;
          }
          waited += GTM_POLL_MS;
          if (waited >= GTM_LOAD_WAIT_MS) finish();
        }, GTM_POLL_MS);
      }

      dataLayer.push({
        event: ACCOUNT_CREATED_EVENT,
        account_id: accountID,
        ...(hashedEmail && {
          user_data: { sha256_email_address: hashedEmail },
        }),
        eventCallback: finish,
        eventTimeout: timeoutMs,
      });
    });

    // Only count the account as tracked if GTM was there to receive the event.
    // Otherwise leave it unmarked so a later visit can retry.
    if (w.google_tag_manager) markTracked(accountID);
  } finally {
    inFlight.delete(accountID);
  }
}
