export const ACCOUNT_CREATED_EVENT = 'account_created';
export const ACCOUNT_CREATED_TIMEOUT_MS = 2000;

const GTM_LOAD_WAIT_MS = 1000;
const inFlight = new Set<string>();

type GTMWindow = Window & {
  dataLayer?: unknown[];
  google_tag_manager?: unknown;
};

export const normalizeEmail = (email: string): string => {
  const normalized = email.trim().toLowerCase();
  const at = normalized.lastIndexOf('@');
  if (at < 0) return normalized;
  const domain = normalized.slice(at + 1);
  if (domain !== 'gmail.com' && domain !== 'googlemail.com') return normalized;
  return `${normalized
    .slice(0, at)
    .split('+')[0]
    .replace(/\./g, '')}@${domain}`;
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

const waitForGTM = (w: GTMWindow, signal: AbortSignal): Promise<boolean> => {
  if (signal.aborted || w.google_tag_manager) {
    return Promise.resolve(!signal.aborted && Boolean(w.google_tag_manager));
  }
  return new Promise((resolve) => {
    const finish = () => {
      clearInterval(poll);
      clearTimeout(timeout);
      signal.removeEventListener('abort', finish);
      resolve(!signal.aborted && Boolean(w.google_tag_manager));
    };
    const poll = setInterval(() => {
      if (w.google_tag_manager) finish();
    }, 50);
    const timeout = setTimeout(finish, GTM_LOAD_WAIT_MS);
    signal.addEventListener('abort', finish, { once: true });
  });
};

type TrackAccountCreatedInput = {
  accountID: string | null | undefined;
  email: string | null | undefined;
  claimConversion: (signal: AbortSignal) => Promise<boolean>;
  timeoutMs?: number;
};

export const trackAccountCreated = async ({
  accountID,
  email,
  claimConversion,
  timeoutMs = ACCOUNT_CREATED_TIMEOUT_MS,
}: TrackAccountCreatedInput): Promise<void> => {
  if (typeof window === 'undefined' || !accountID || inFlight.has(accountID))
    return;
  inFlight.add(accountID);

  return new Promise<void>((resolve) => {
    const controller = new AbortController();
    const { signal } = controller;
    const finish = () => {
      if (signal.aborted) return;
      clearTimeout(timeout);
      controller.abort();
      inFlight.delete(accountID);
      resolve();
    };
    // Bound the claim and hashing too, so tracking cannot hold up signup.
    const timeout = setTimeout(finish, timeoutMs);

    const send = async () => {
      const w = window as GTMWindow;
      if (!(await waitForGTM(w, signal))) return finish();
      const hashedEmail = email
        ? await sha256Hex(normalizeEmail(email)).catch(() => undefined)
        : undefined;
      if (signal.aborted) return;
      if (!(await claimConversion(signal)) || signal.aborted) return finish();

      w.dataLayer = w.dataLayer || [];
      w.dataLayer.push({
        event: ACCOUNT_CREATED_EVENT,
        account_id: accountID,
        ...(hashedEmail && {
          user_data: { sha256_email_address: hashedEmail },
        }),
        eventCallback: finish,
        eventTimeout: timeoutMs,
      });
    };
    void send().catch(finish);
  });
};
