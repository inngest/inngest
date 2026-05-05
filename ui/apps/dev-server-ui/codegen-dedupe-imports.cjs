/**
 * Post-processes a graphql-codegen output:
 *
 *   1. Dedupes `import type { ... } from '...';` lines that share a source
 *      path. The typescript and typescript-operations plugins both emit
 *      scalar-mapped imports (e.g.
 *      `import type { SpanMetadataKind } from '@inngest/components/...';`)
 *      when they share an output file. With `useTypeImports: true` this
 *      causes "Duplicate identifier" errors under TS strict mode. The hook
 *      merges all imports of the same source into a single statement and
 *      drops duplicate symbols.
 *
 *   2. Strips a self-referential namespace import the typescript-operations
 *      plugin emits when `importSchemaTypesFrom` points at the same file
 *      it is generating (we use this to suppress its enum/input emission
 *      so they do not duplicate the typescript plugin's). The import looks
 *      like `import type * as Types from './generated-types';` inside
 *      `generated-types.ts` itself; TS allows it but it triggers
 *      noUnusedLocals warnings and reads as obviously circular. We rewrite
 *      `Types.<name>` references back to bare `<name>` and remove the
 *      import line.
 */
const fs = require('fs');
const path = require('path');

const filePath = process.argv[2];
if (!filePath) {
  process.exit(0);
}

const absPath = path.resolve(filePath);
let src;
try {
  src = fs.readFileSync(absPath, 'utf8');
} catch {
  process.exit(0);
}

// (2) Strip the self-referential namespace import emitted by
// typescript-operations when importSchemaTypesFrom points at this same file.
const fileBaseName = path.basename(absPath).replace(/\.[cm]?[jt]sx?$/, '');
const selfImportRe = new RegExp(
  String.raw`^import type \* as (\w+) from ['"]\.\/${fileBaseName}['"];\s*$`,
  'm',
);
const selfImportMatch = src.match(selfImportRe);
if (selfImportMatch) {
  const ns = selfImportMatch[1];
  src = src.replace(selfImportRe, '');
  // Replace `Types.Foo` with `Foo`. The namespace alias is a TS identifier so
  // a word-boundary match is sufficient here.
  src = src.replace(new RegExp(String.raw`\b${ns}\.`, 'g'), '');
}

const importRe =
  /^import type \{\s*([^}]+?)\s*\} from ['"]([^'"]+)['"];\s*$/gm;
const bySource = new Map();
const order = [];
for (const match of src.matchAll(importRe)) {
  const source = match[2];
  const symbols = match[1]
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);
  if (!bySource.has(source)) {
    bySource.set(source, new Set());
    order.push(source);
  }
  const set = bySource.get(source);
  for (const sym of symbols) {
    set.add(sym);
  }
}

if (order.length === 0) {
  fs.writeFileSync(absPath, src);
  process.exit(0);
}

const withoutImports = src.replace(importRe, '');
const merged = order
  .map((source) => {
    const symbols = Array.from(bySource.get(source)).sort();
    return `import type { ${symbols.join(', ')} } from '${source}';`;
  })
  .join('\n');

const cleaned = withoutImports.replace(/^\s*\n+/, '');
fs.writeFileSync(absPath, `${merged}\n\n${cleaned}`);
