import { copyFileSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'fs';
import { join, resolve } from 'path';
import { generateFiles } from 'fumadocs-openapi';
import { createOpenAPI } from 'fumadocs-openapi/server';
import { parse as parseYaml } from 'yaml';

const appRoot = resolve(import.meta.dirname, '..');
const repoRoot = resolve(appRoot, '../../..');

type OpenAPIDocument = {
  paths?: Record<string, Record<string, { requestBody?: { content?: Record<string, unknown> } }>>;
};

/**
 * Remove requestBody entries with empty content — some Stoplight-generated specs
 * include `requestBody: { content: {} }` on GET endpoints, which fumadocs-openapi
 * doesn't handle.
 */
function cleanSpec(doc: OpenAPIDocument): OpenAPIDocument {
  const methods = ['get', 'post', 'put', 'patch', 'delete', 'head', 'options', 'trace'];
  for (const pathItem of Object.values(doc.paths ?? {})) {
    for (const method of methods) {
      const op = pathItem[method];
      if (op?.requestBody && Object.keys(op.requestBody.content ?? {}).length === 0) {
        delete op.requestBody;
      }
    }
  }
  return doc;
}

async function main() {
  const v1SpecPath = resolve(repoRoot, 'docs/openapi/v3/api/v1/spec.yaml');
  const v2SpecPath = resolve(repoRoot, 'docs/openapi/v3/api/v2/service.swagger.json');

  // Copy specs to public/ so they can be served for the API playground
  const publicSpecsDir = join(appRoot, 'public/api-specs');
  mkdirSync(publicSpecsDir, { recursive: true });
  copyFileSync(v1SpecPath, join(publicSpecsDir, 'v1.yaml'));
  copyFileSync(v2SpecPath, join(publicSpecsDir, 'v2.json'));
  console.log('Copied specs to public/api-specs/');

  const v1Doc = cleanSpec(parseYaml(readFileSync(v1SpecPath, 'utf8')) as OpenAPIDocument);
  const v2Doc = JSON.parse(readFileSync(v2SpecPath, 'utf8')) as OpenAPIDocument;

  // Also write v1 as JSON so the browser APIPage wrapper can fetch it without YAML parsing
  writeFileSync(join(publicSpecsDir, 'v1.json'), JSON.stringify(v1Doc, null, 2));
  console.log('Wrote public/api-specs/v1.json');

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
