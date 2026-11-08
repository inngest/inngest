// ResolvedPathSegment is one step of a *concrete* path into a parsed JSON
// value -- a real array index or a real object key, no wildcards (unlike
// PathSegmentLike in pathHints.ts, which is the *declared* path pattern a
// PathHint carries, wildcards included). collectAtPath (pathHints.ts)
// resolves a declared pattern against a concrete value, producing one of
// these per match.
export type ResolvedPathSegment = string | number;

export type JsonSpans = {
  // The pretty-printed text itself -- byte-identical to
  // `JSON.stringify(value, null, 2)` (verified by construction: this
  // walks the exact same object/array/string/number/boolean/null cases
  // in the exact same order and separators V8's own formatter does).
  text: string;
  // One entry per node ever visited (object, array, or leaf), keyed by
  // JSON.stringify(path) -- the offset range (into `text`) of that
  // node's own rendered value, closing brace/bracket included for an
  // object/array.
  valueRange: Map<string, [number, number]>;
  // One entry per *object property* ever visited, keyed the same way as
  // valueRange (by the property's own path, not its parent's) -- the
  // offset range of that property's key token alone (quotes included,
  // not the trailing ": ").
  keyRange: Map<string, [number, number]>;
};

export function resolvedPathKey(path: readonly ResolvedPathSegment[]): string {
  return JSON.stringify(path);
}

// stringifyWithSpans pretty-prints value exactly like
// `JSON.stringify(value, null, 2)`, while recording every node's own
// text range as it goes -- so a caller holding a *path* into the same
// value (from collectAtPath) can look up the exact substring that path
// rendered to, rather than searching the output text for it. This is
// what lets two structurally distinct occurrences of the identical
// string value (e.g. an unrelated field that happens to equal a nearby
// session ID) resolve to the correct one: the search is by structural
// position, never by literal text content.
export function stringifyWithSpans(value: unknown): JsonSpans {
  let text = '';
  const valueRange = new Map<string, [number, number]>();
  const keyRange = new Map<string, [number, number]>();

  function write(
    v: unknown,
    path: ResolvedPathSegment[],
    indent: string,
  ): void {
    const start = text.length;

    if (Array.isArray(v)) {
      if (v.length === 0) {
        text += '[]';
      } else {
        text += '[\n';
        const childIndent = indent + '  ';
        v.forEach((item, i) => {
          text += childIndent;
          write(item, [...path, i], childIndent);
          text += i < v.length - 1 ? ',\n' : '\n';
        });
        text += indent + ']';
      }
    } else if (v !== null && typeof v === 'object') {
      const entries = Object.entries(v as Record<string, unknown>);
      if (entries.length === 0) {
        text += '{}';
      } else {
        text += '{\n';
        const childIndent = indent + '  ';
        entries.forEach(([k, val], i) => {
          text += childIndent;
          const keyStart = text.length;
          text += JSON.stringify(k);
          keyRange.set(resolvedPathKey([...path, k]), [keyStart, text.length]);
          text += ': ';
          write(val, [...path, k], childIndent);
          text += i < entries.length - 1 ? ',\n' : '\n';
        });
        text += indent + '}';
      }
    } else {
      text += JSON.stringify(v);
    }

    valueRange.set(resolvedPathKey(path), [start, text.length]);
  }

  write(value, [], '');
  return { text, valueRange, keyRange };
}
