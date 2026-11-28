import { describe, expect, it } from 'vitest';

import { canonicalMetadataKind, hasMetadataKind } from './metadataKind';

describe('canonicalMetadataKind', () => {
  it.each([
    // Full kinds, w/o isUser (the cloud API) and w/ it (the cqrs read path).
    [{ kind: 'inngest.score' }, 'inngest.score'],
    [{ kind: 'userland.foo' }, 'userland.foo'],
    [{ kind: 'inngest.score', isUser: false }, 'inngest.score'],
    [{ kind: 'userland.foo', isUser: true }, 'userland.foo'],
    // Prefix-stripped kinds from the DuckDB read path.
    [{ kind: 'score', isUser: false }, 'inngest.score'],
    [{ kind: 'score.accuracy', isUser: false }, 'inngest.score.accuracy'],
    [{ kind: 'foo', isUser: true }, 'userland.foo'],
    [{ kind: 'score', isUser: null }, 'inngest.score'],
    // A stripped user kind that itself starts w/ the internal prefix.
    [{ kind: 'inngest.score', isUser: true }, 'userland.inngest.score'],
  ])('%o -> %s', (md, want) => {
    expect(canonicalMetadataKind(md)).toBe(want);
  });

  it('keeps a user and an internal kind w/ the same stripped name apart', () => {
    expect(hasMetadataKind({ kind: 'ai', isUser: false }, 'inngest.ai')).toBe(true);
    expect(hasMetadataKind({ kind: 'ai', isUser: true }, 'inngest.ai')).toBe(false);
  });
});
