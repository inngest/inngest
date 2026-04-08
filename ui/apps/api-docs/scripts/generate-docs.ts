import { copyFileSync, mkdirSync, readFileSync, writeFileSync } from 'fs';
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

  console.log('\nGenerating v1 API docs...');
  await generateFiles({
    input: openapiV1,
    output: join(appRoot, 'content/docs/v1'),
    per: 'operation',
  });

  console.log('\nGenerating v2 API docs...');
  await generateFiles({
    input: openapiV2,
    output: join(appRoot, 'content/docs/v2'),
    per: 'operation',
  });

  console.log('\nDone.');
}

main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});
