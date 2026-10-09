import { describe, expect, it } from 'vitest';

import { GetRunTraceDocument, TraceDetailsFragmentDoc } from './graphql';

type Definition = {
  kind: string;
  name?: { value: string };
  selectionSet?: { selections: { kind: string; name?: { value: string } }[] };
};

function traceDetailsFields(document: { definitions: readonly unknown[] }) {
  const fragment = (document.definitions as Definition[]).find((definition) => {
    return definition.kind === 'FragmentDefinition' && definition.name?.value === 'TraceDetails';
  });

  return (fragment?.selectionSet?.selections ?? []).map((selection) => {
    return selection.name?.value;
  });
}

describe('generated run trace document', () => {
  it('selects the same trace fields as the fragment it inlines', () => {
    const fields = traceDetailsFields(GetRunTraceDocument);

    expect(fields).toContain('groupKind');
    expect(fields).toContain('origin');
    expect(fields).toEqual(traceDetailsFields(TraceDetailsFragmentDoc));
  });
});
