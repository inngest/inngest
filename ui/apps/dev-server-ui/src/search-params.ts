import { defaultParseSearch } from '@tanstack/react-router';

export function parseDevServerSearch(searchStr: string) {
  const parsed = defaultParseSearch(searchStr) as Record<string, unknown>;

  // The Functions filter is text. Preserve its original spelling before the
  // router normalizes unquoted numeric and boolean values.
  if (typeof parsed.query === 'number' || typeof parsed.query === 'boolean') {
    const query = new URLSearchParams(searchStr).get('query');
    if (query !== null) {
      parsed.query = query;
    }
  }

  return parsed;
}
