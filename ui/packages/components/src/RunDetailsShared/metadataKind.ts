const KindPrefixInngest = 'inngest.';
const KindPrefixUserland = 'userland.';

export type MetadataKindRef = {
  kind: string;
  // Optional: only APIs that can return prefix-stripped kinds send it.
  isUser?: boolean | null;
};

// Span metadata kinds arrive either full (`inngest.score`, `userland.foo`) or,
// from the DuckDB trace store, w/o their prefix (`score`, `foo`) plus isUser.
// This returns the full kind either way, so match kinds through it.
export function canonicalMetadataKind({ kind, isUser }: MetadataKindRef): string {
  if (isUser === undefined || isUser === null) {
    if (kind.startsWith(KindPrefixInngest) || kind.startsWith(KindPrefixUserland)) {
      return kind;
    }
    return `${KindPrefixInngest}${kind}`;
  }

  // A full kind always carries the prefix isUser implies, so any other kind
  // was stripped, even one that happens to start w/ the other prefix.
  const prefix = isUser ? KindPrefixUserland : KindPrefixInngest;
  return kind.startsWith(prefix) ? kind : `${prefix}${kind}`;
}

export function hasMetadataKind(md: MetadataKindRef, kind: string): boolean {
  return canonicalMetadataKind(md) === kind;
}
