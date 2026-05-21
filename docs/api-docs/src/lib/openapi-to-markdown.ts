import { createOpenAPI } from 'fumadocs-openapi/server';
import { stringify as stringifyYaml } from 'yaml';

import { generateCurl } from './openapi-curl';
import { reduceSpec } from './openapi-spec-reducer';

// Avoid `node:path` here — vite.config aliases `node:path` to a minimal browser
// shim (src/shims/node-path.ts) that does not export `resolve`. The current
// file's absolute directory is enough to locate the public/ specs.
const PUBLIC_DIR = `${import.meta.dirname}/../../public`;

/**
 * Map the `document` prop emitted into the generated MDX (e.g. `/api-specs/v1.json`)
 * to the on-disk path of the same spec. The MDX-side keys are public-URL paths
 * because the browser-side APIPage fetches them at runtime; on the server we
 * read the same files directly from the public/ directory.
 */
const SPEC_PATHS: Record<string, string> = {
  '/api-specs/v1.json': `${PUBLIC_DIR}/api-specs/v1.json`,
  '/api-specs/v2.json': `${PUBLIC_DIR}/api-specs/v2.json`,
};

const server = createOpenAPI({
  input: async () => SPEC_PATHS,
});

/**
 * Render the operation as a `## Usage` + `## OpenAPI` pair, mirroring the
 * Mintlify `.md` format. Returns an empty string when the spec or operation
 * isn't found — callers fall back to the page's processed-markdown body.
 *
 * The `## OpenAPI` block is a self-contained reduced spec (paths + only the
 * components transitively referenced by this operation), which gives LLMs an
 * exact, parseable contract rather than a hand-formatted approximation.
 */
export async function renderOperationMarkdown(
  documentKey: string,
  path: string,
  method: string
): Promise<string> {
  if (!(documentKey in SPEC_PATHS)) return '';

  // `bundled` preserves $refs so the reduced spec stays compact and uses the
  // same component names as the source. `dereferenced` would inline every
  // schema and bloat the output.
  const schema = await server.getSchema(documentKey);
  const doc = schema.bundled as Parameters<typeof reduceSpec>[0];
  if (!doc.paths?.[path]?.[method.toLowerCase()]) return '';

  const reduced = reduceSpec(doc, path, method);
  if (!reduced) return '';

  // Both helpers walk the same OpenAPI doc with their own narrowed views of
  // the spec — they only read the fields they care about.
  const curl = generateCurl(doc as Parameters<typeof generateCurl>[0], path, method);
  const yaml = stringifyYaml(reduced, { lineWidth: 0 }).trimEnd();
  const fenceInfo = `yaml ${documentKey} ${method.toUpperCase()} ${path}`;

  const out: string[] = [];
  if (curl) {
    out.push('## Usage', '', '```bash', curl, '```', '');
  }
  out.push('## OpenAPI', '', `\`\`\`\`${fenceInfo}`, yaml, '````');
  return out.join('\n');
}
