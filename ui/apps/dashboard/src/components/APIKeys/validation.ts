export const API_KEY_NAME_MAX = 128;

export function validateRestrictedAPIKey(input: {
  name: string;
  allEnvironments: boolean;
  workspaceID?: string;
  permissions: string[];
}) {
  return {
    name: validateAPIKeyName(input.name),
    environment:
      !input.allEnvironments && !input.workspaceID
        ? 'Select an environment.'
        : null,
    permissions:
      input.permissions.length === 0 ? 'Select at least one permission.' : null,
  };
}

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
