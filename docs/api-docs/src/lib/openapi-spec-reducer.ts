/**
 * Reduce a full OpenAPI document down to just one operation plus the
 * components it transitively references. The bundled spec (with `$ref`s
 * intact) is what we want to embed in the per-page markdown — it stays
 * compact and is easy for an LLM to consume.
 */

type JSONValue = string | number | boolean | null | JSONValue[] | { [k: string]: JSONValue };
type OASDocument = {
  openapi?: string;
  swagger?: string;
  info?: unknown;
  servers?: unknown;
  security?: Array<Record<string, string[]>>;
  tags?: Array<{ name: string; [k: string]: unknown }>;
  paths?: Record<string, Record<string, unknown> & { parameters?: unknown }>;
  components?: Record<string, Record<string, unknown>>;
  [k: string]: unknown;
};

export function reduceSpec(
  doc: OASDocument,
  path: string,
  method: string
): OASDocument | null {
  const pathItem = doc.paths?.[path];
  const methodLower = method.toLowerCase();
  const operation = pathItem?.[methodLower];
  if (!pathItem || !operation) return null;

  const reducedPathItem: Record<string, unknown> = { [methodLower]: operation };
  if (pathItem.parameters) reducedPathItem.parameters = pathItem.parameters;

  const refsToInclude = new Set<string>();
  collectRefs(operation, refsToInclude);
  if (pathItem.parameters) collectRefs(pathItem.parameters, refsToInclude);

  // Expand transitively — every component we pull in may reference others.
  const visited = new Set<string>();
  const components: Record<string, Record<string, unknown>> = {};
  while (refsToInclude.size > visited.size) {
    for (const ref of refsToInclude) {
      if (visited.has(ref)) continue;
      visited.add(ref);
      const target = resolveRef(doc, ref);
      if (!target) continue;
      // Refs look like `#/components/schemas/Foo` → split is
      // ['#', 'components', 'schemas', 'Foo']. We want the group ('schemas')
      // and name ('Foo').
      const parts = ref.split('/');
      const group = parts[2];
      const name = parts[3];
      if (!group || !name) continue;
      components[group] ??= {};
      components[group][name] = target as Record<string, unknown>;
      collectRefs(target, refsToInclude);
    }
  }

  // Security schemes are referenced by name in `security` arrays, not via $ref.
  // Pick up any scheme used by the operation or inherited from the document.
  const security = (operation as { security?: Array<Record<string, string[]>> }).security
    ?? doc.security;
  const securitySchemes = doc.components?.securitySchemes as
    | Record<string, unknown>
    | undefined;
  if (security?.length && securitySchemes) {
    for (const req of security) {
      for (const name of Object.keys(req)) {
        if (securitySchemes[name]) {
          components.securitySchemes ??= {};
          components.securitySchemes[name] = securitySchemes[name] as Record<string, unknown>;
        }
      }
    }
  }

  // Limit `tags` to those actually used by the operation.
  const opTags = (operation as { tags?: string[] }).tags ?? [];
  const tags = doc.tags?.filter((t) => opTags.includes(t.name));

  const reduced: OASDocument = {};
  if (doc.openapi) reduced.openapi = doc.openapi;
  else if (doc.swagger) reduced.swagger = doc.swagger;
  if (doc.info) reduced.info = doc.info;
  if (doc.servers) reduced.servers = doc.servers;
  if (doc.security) reduced.security = doc.security;
  if (tags?.length) reduced.tags = tags;
  reduced.paths = { [path]: reducedPathItem };
  if (Object.keys(components).length) reduced.components = components;
  return reduced;
}

function collectRefs(node: unknown, out: Set<string>): void {
  if (node === null || typeof node !== 'object') return;
  if (Array.isArray(node)) {
    for (const item of node) collectRefs(item, out);
    return;
  }
  const obj = node as Record<string, unknown>;
  if (typeof obj.$ref === 'string' && obj.$ref.startsWith('#/components/')) {
    out.add(obj.$ref);
  }
  for (const value of Object.values(obj)) collectRefs(value, out);
}

function resolveRef(doc: OASDocument, ref: string): JSONValue | undefined {
  // Only handle local `#/...` refs — anything external should already have
  // been bundled by `processDocument()` upstream.
  if (!ref.startsWith('#/')) return undefined;
  const parts = ref.slice(2).split('/').map(decodePointer);
  let cur: unknown = doc;
  for (const part of parts) {
    if (cur === null || typeof cur !== 'object') return undefined;
    cur = (cur as Record<string, unknown>)[part];
  }
  return cur as JSONValue | undefined;
}

function decodePointer(segment: string): string {
  // RFC 6901: `~1` → `/`, `~0` → `~` (must be in that order).
  return segment.replace(/~1/g, '/').replace(/~0/g, '~');
}
