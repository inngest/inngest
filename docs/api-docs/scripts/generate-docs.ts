import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'fs';
import { join, resolve } from 'path';
import { generateFiles } from 'fumadocs-openapi';
import { createOpenAPI } from 'fumadocs-openapi/server';
import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';

const appRoot = resolve(import.meta.dirname, '..');
const repoRoot = resolve(appRoot, '../..');

type Operation = {
  requestBody?: { content?: Record<string, unknown> };
  tags?: string[];
  [extension: `x-${string}`]: unknown;
};

type Tag = { name: string; 'x-displayName'?: string; summary?: string };

type OpenAPIDocument = {
  paths?: Record<string, Record<string, Operation>>;
  tags?: Tag[];
};

const HTTP_METHODS = ['get', 'post', 'put', 'patch', 'delete', 'head', 'options', 'trace'];

/**
 * Operations tagged with any of these names are stripped from the spec before
 * doc generation. Use the "Internal" tag in the proto/spec to hide an endpoint
 * from the public docs nav and the API playground.
 */
const HIDDEN_TAGS = new Set(['Internal']);

function hideTaggedOperations(doc: OpenAPIDocument): OpenAPIDocument {
  for (const [path, pathItem] of Object.entries(doc.paths ?? {})) {
    for (const method of HTTP_METHODS) {
      const op = pathItem[method];
      if (op?.tags?.some((t) => HIDDEN_TAGS.has(t))) {
        delete pathItem[method];
      }
    }
    if (Object.keys(pathItem).length === 0) delete doc.paths![path];
  }
  if (doc.tags) doc.tags = doc.tags.filter((t) => !HIDDEN_TAGS.has(t.name));
  return doc;
}

/**
 * "Soft" tags that don't create a nav group. When an operation is tagged with
 * one of these, the tag is removed from the operation and the corresponding
 * `x-*` extension is set so renderers (e.g. our APIPage wrapper) can show a
 * badge while the operation stays grouped under its primary tag.
 *
 * Add new entries here to introduce new badges (e.g. Preview → x-preview).
 */
const SOFT_TAGS: Record<string, `x-${string}`> = {
  Beta: 'x-beta',
};

function applySoftTags(doc: OpenAPIDocument): OpenAPIDocument {
  for (const pathItem of Object.values(doc.paths ?? {})) {
    for (const method of HTTP_METHODS) {
      const op = pathItem[method];
      if (!op?.tags) continue;
      const remaining: string[] = [];
      for (const t of op.tags) {
        const ext = SOFT_TAGS[t];
        if (ext) op[ext] = true;
        else remaining.push(t);
      }
      op.tags = remaining;
    }
  }
  if (doc.tags) doc.tags = doc.tags.filter((t) => !(t.name in SOFT_TAGS));
  return doc;
}

/**
 * fumadocs derives a tag's display name by running `idToTitle` on the tag name,
 * which inserts spaces before uppercase letters. That mangles already-spaced
 * names like "Partner API" into "Partner A P I". Setting `x-displayName` on the
 * tag short-circuits that and is returned verbatim.
 */
function preserveSpacedTagDisplayNames(doc: OpenAPIDocument): OpenAPIDocument {
  for (const tag of doc.tags ?? []) {
    if (tag.name.includes(' ') && !tag['x-displayName']) {
      tag['x-displayName'] = tag.name;
    }
  }
  return doc;
}

/**
 * Remove requestBody entries with empty content — some Stoplight-generated specs
 * include `requestBody: { content: {} }` on GET endpoints, which fumadocs-openapi
 * doesn't handle.
 */
function cleanSpec(doc: OpenAPIDocument): OpenAPIDocument {
  for (const pathItem of Object.values(doc.paths ?? {})) {
    for (const method of HTTP_METHODS) {
      const op = pathItem[method];
      if (op?.requestBody && Object.keys(op.requestBody.content ?? {}).length === 0) {
        delete op.requestBody;
      }
    }
  }
  return doc;
}

/**
 * fumadocs-openapi's `groupBy: 'tag'` looks up each operation's tag in the
 * top-level `tags` array. If a referenced tag isn't declared there,
 * `builder.fromTagName(tag)` returns undefined and generation crashes with an
 * opaque `Cannot destructure property 'displayName' of ...` error. Validate up
 * front and report exactly which tag is missing on which operation.
 */
function assertTagsDeclared(doc: OpenAPIDocument, specLabel: string): void {
  const declared = new Set((doc.tags ?? []).map((t) => t.name));
  const missing: { tag: string; method: string; path: string }[] = [];
  for (const [path, pathItem] of Object.entries(doc.paths ?? {})) {
    for (const method of HTTP_METHODS) {
      const op = pathItem[method];
      for (const tag of op?.tags ?? []) {
        if (!declared.has(tag)) missing.push({ tag, method: method.toUpperCase(), path });
      }
    }
  }
  if (missing.length === 0) return;
  const lines = missing.map((m) => `  - ${m.method} ${m.path} references tag "${m.tag}"`);
  const unique = [...new Set(missing.map((m) => m.tag))];
  throw new Error(
    `${specLabel}: ${missing.length} operation(s) reference tags not declared in the top-level "tags" array.\n` +
      lines.join('\n') +
      `\n\nDeclare these tags at the top level of the spec: ${unique
        .map((t) => `"${t}"`)
        .join(', ')}.\n` +
      `For the v2 API this means adding entries to the swagger \`tags:\` block in proto/api/v2/service.proto and re-running \`make docs\`.`
  );
}

async function main() {
  const v1SpecPath = resolve(repoRoot, 'docs/openapi/v3/api/v1/spec.yaml');
  const v2SpecPath = resolve(repoRoot, 'docs/openapi/v3/api/v2/service.swagger.json');

  const v1Doc = preserveSpacedTagDisplayNames(
    applySoftTags(
      hideTaggedOperations(
        cleanSpec(parseYaml(readFileSync(v1SpecPath, 'utf8')) as OpenAPIDocument)
      )
    )
  );
  const v2Doc = preserveSpacedTagDisplayNames(
    applySoftTags(
      hideTaggedOperations(JSON.parse(readFileSync(v2SpecPath, 'utf8')) as OpenAPIDocument)
    )
  );

  assertTagsDeclared(v1Doc, 'v1 spec');
  assertTagsDeclared(v2Doc, 'v2 spec');

  // Write filtered specs to public/ so the API playground sees the same view as the docs.
  const publicSpecsDir = join(appRoot, 'public/api-specs');
  mkdirSync(publicSpecsDir, { recursive: true });
  writeFileSync(join(publicSpecsDir, 'v1.yaml'), stringifyYaml(v1Doc));
  writeFileSync(join(publicSpecsDir, 'v1.json'), JSON.stringify(v1Doc, null, 2));
  writeFileSync(join(publicSpecsDir, 'v2.json'), JSON.stringify(v2Doc, null, 2));
  console.log('Wrote public/api-specs/');

  // Separate instances so generation stays isolated per version.
  // URL keys ensure generated MDX references /api-specs/... not absolute FS paths.
  const openapiV1 = createOpenAPI({
    input: async () => ({ '/api-specs/v1.json': v1Doc }),
  });

  const openapiV2 = createOpenAPI({
    input: async () => ({ '/api-specs/v2.json': v2Doc }),
  });

  // Clean generated v1 content, preserving hand-written files.
  const v1Dir = join(appRoot, 'content/docs/v1');
  const v1IndexPath = join(v1Dir, 'index.mdx');
  const v1IndexContent = readFileSync(v1IndexPath, 'utf8');
  rmSync(v1Dir, { recursive: true, force: true });
  mkdirSync(v1Dir, { recursive: true });
  writeFileSync(v1IndexPath, v1IndexContent);

  console.log('\nGenerating v1 API docs...');
  await generateFiles({
    input: openapiV1,
    output: v1Dir,
    per: 'operation',
    groupBy: 'tag',
    meta: { groupStyle: 'separator' },
    beforeWrite(files) {
      const topMeta = files.find((f) => f.path === 'meta.json');
      if (topMeta) {
        const meta = JSON.parse(topMeta.content);
        meta.title = 'v1';
        meta.root = false;
        topMeta.content = JSON.stringify(meta, null, 2);
      }
    },
  });

  // Clean generated v2 content so stale files don't accumulate between runs.
  rmSync(join(appRoot, 'content/docs/v2'), { recursive: true, force: true });
  mkdirSync(join(appRoot, 'content/docs/v2'), { recursive: true });

  console.log('\nGenerating v2 API docs...');
  await generateFiles({
    input: openapiV2,
    output: join(appRoot, 'content/docs/v2'),
    per: 'operation',
    groupBy: 'tag',
    meta: { groupStyle: 'separator' },
    beforeWrite(files) {
      // Strip the "V2_" prefix from generated file paths and their meta.json references.
      for (const file of files) {
        file.path = file.path.replace(/V2_/g, '');
      }
      // Update page references inside meta.json files to match renamed paths.
      for (const file of files) {
        if (file.path.endsWith('meta.json')) {
          const meta = JSON.parse(file.content);
          if (Array.isArray(meta.pages)) {
            meta.pages = meta.pages.map((p: string) => p.replace(/V2_/g, ''));
          }
          file.content = JSON.stringify(meta, null, 2);
        }
      }
      // Set the nav heading for v2.
      const topMeta = files.find((f) => f.path === 'meta.json');
      if (topMeta) {
        const meta = JSON.parse(topMeta.content);
        meta.title = 'v2';
        topMeta.content = JSON.stringify(meta, null, 2);
      }
    },
  });

  console.log('\nDone.');
}

main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});
