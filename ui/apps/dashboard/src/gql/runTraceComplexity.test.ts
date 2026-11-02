import { Kind, type DocumentNode, type SelectionSetNode } from 'graphql';
import { formatDocument } from 'urql';
import { describe, expect, it } from 'vitest';

import { GetRunTraceDocument, GetRunTraceTicksDocument } from './graphql';

/** Cloud rejects any query that costs more than this. */
const CLOUD_COMPLEXITY_LIMIT = 300;

/**
 * Costs a query the way Cloud does: one point per field, whatever it holds,
 * with fragments counted where they are spread. The urql cache adds a
 * `__typename` to every selection set before the query is sent, so those count
 * too.
 */
function complexity(document: DocumentNode) {
  const sent = formatDocument(document);
  const fragments = new Map<string, SelectionSetNode>();
  for (const definition of sent.definitions) {
    if (definition.kind === Kind.FRAGMENT_DEFINITION) {
      fragments.set(definition.name.value, definition.selectionSet);
    }
  }

  function selectionSetCost(selectionSet: SelectionSetNode): number {
    let cost = 0;

    for (const selection of selectionSet.selections) {
      if (selection.kind === Kind.FIELD) {
        cost +=
          1 +
          (selection.selectionSet
            ? selectionSetCost(selection.selectionSet)
            : 0);
      } else if (selection.kind === Kind.INLINE_FRAGMENT) {
        cost += selectionSetCost(selection.selectionSet);
      } else {
        const fragment = fragments.get(selection.name.value);
        cost += fragment ? selectionSetCost(fragment) : 0;
      }
    }

    return cost;
  }

  let cost = 0;
  for (const definition of sent.definitions) {
    if (definition.kind === Kind.OPERATION_DEFINITION) {
      cost += selectionSetCost(definition.selectionSet);
    }
  }

  return cost;
}

describe('run trace queries', () => {
  // The trace query goes five levels deep and has little room left: each field
  // added to its TraceDetails fragment costs five times over. Anything that
  // needs more fields, such as span ticks, belongs in its own query.
  it('stay under the complexity limit Cloud enforces', () => {
    expect(complexity(GetRunTraceDocument)).toBeLessThanOrEqual(
      CLOUD_COMPLEXITY_LIMIT,
    );
    expect(complexity(GetRunTraceTicksDocument)).toBeLessThanOrEqual(
      CLOUD_COMPLEXITY_LIMIT,
    );
  });
});
