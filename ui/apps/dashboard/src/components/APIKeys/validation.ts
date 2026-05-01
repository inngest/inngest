export const API_KEY_NAME_MAX = 128;

// Returns a user-facing error string, or null when the name is valid.
// Trims whitespace and enforces the 1..128 length contract shared with the
// backend (see monorepo api-key-spec.md §3).
export function validateAPIKeyName(raw: string): string | null {
  const trimmed = raw.trim();
  if (!trimmed) return 'Name is required.';
  if (trimmed.length > API_KEY_NAME_MAX) {
    return `Name must be ${API_KEY_NAME_MAX} characters or fewer.`;
  }
  return null;
}
