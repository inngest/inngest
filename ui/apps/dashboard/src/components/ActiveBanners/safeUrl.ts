// Schemes we allow on an admin-authored banner CTA. Banners render to
// authenticated users and an admin-supplied `javascript:` or `data:` href
// would be a stored-XSS primitive, so we validate before passing the URL to
// an <a>.
const allowedSchemes = new Set(['http:', 'https:', 'mailto:']);

export function isSafeCTAURL(raw: string): boolean {
  if (!raw) return false;

  // Allow site-relative paths (e.g. "/docs/upgrading"). These have no scheme
  // and cannot express a dangerous one.
  if (raw.startsWith('/') && !raw.startsWith('//')) return true;

  try {
    // Any base is fine here: we only care whether the parsed protocol is in
    // the allow list. A malformed URL throws and is rejected.
    const parsed = new URL(raw);
    return allowedSchemes.has(parsed.protocol);
  } catch {
    return false;
  }
}
