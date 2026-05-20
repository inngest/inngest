/**
 * Generate a `curl` example for a single OpenAPI operation. The output is
 * deliberately a single, well-formatted bash command suitable for dropping
 * into a `## Usage` section.
 */

type OASSchema = {
  type?: string | string[];
  format?: string;
  enum?: unknown[];
  default?: unknown;
  example?: unknown;
  examples?: unknown[];
  properties?: Record<string, OASSchema>;
  required?: string[];
  items?: OASSchema;
  oneOf?: OASSchema[];
  anyOf?: OASSchema[];
  allOf?: OASSchema[];
  $ref?: string;
};
type OASParameter = {
  name: string;
  in: 'path' | 'query' | 'header' | 'cookie';
  required?: boolean;
  example?: unknown;
  schema?: OASSchema;
};
type OASMediaType = { schema?: OASSchema; example?: unknown };
type OASOperation = {
  parameters?: OASParameter[];
  requestBody?: { content?: Record<string, OASMediaType> };
  security?: Array<Record<string, string[]>>;
};
type OASDocument = {
  servers?: Array<{ url: string }>;
  security?: Array<Record<string, string[]>>;
  paths?: Record<
    string,
    Record<string, OASOperation> & { parameters?: OASParameter[] }
  >;
  components?: {
    securitySchemes?: Record<
      string,
      { type?: string; scheme?: string; name?: string; in?: string }
    >;
    schemas?: Record<string, OASSchema>;
  };
};

export function generateCurl(doc: OASDocument, path: string, method: string): string {
  const pathItem = doc.paths?.[path];
  const operation = pathItem?.[method.toLowerCase()];
  if (!pathItem || !operation) return '';

  const allParams: OASParameter[] = [
    ...(pathItem.parameters ?? []),
    ...(operation.parameters ?? []),
  ];
  const filledPath = fillPathParams(path, allParams);
  const baseUrl = (doc.servers?.[0]?.url ?? '').replace(/\/$/, '');
  const queryString = buildQueryString(allParams);
  const url = `${baseUrl}${filledPath}${queryString}`;

  const lines: string[] = [`curl -X ${method.toUpperCase()} "${url}"`];

  const authHeader = authHeaderFor(operation, doc);
  if (authHeader) lines.push(`-H "${authHeader}"`);

  for (const p of allParams) {
    if (p.in === 'header') {
      lines.push(`-H "${p.name}: ${sampleParam(p)}"`);
    }
  }

  const body = operation.requestBody?.content;
  if (body) {
    const [mediaType, media] = Object.entries(body)[0] ?? [];
    if (mediaType) {
      lines.push(`-H "Content-Type: ${mediaType}"`);
      const sample = media.example ?? sampleFromSchema(media.schema, doc);
      const payload = mediaType.includes('json')
        ? JSON.stringify(sample, null, 2)
        : String(sample ?? '');
      lines.push(`-d '${escapeSingleQuotes(payload)}'`);
    }
  }

  return lines.map((line, i) => (i === 0 ? line : `  ${line}`)).join(' \\\n');
}

function fillPathParams(path: string, params: OASParameter[]): string {
  return path.replace(/\{([^}]+)\}/g, (_, name: string) => {
    const param = params.find((p) => p.in === 'path' && p.name === name);
    // Only substitute when there's a real example. Otherwise keep the
    // placeholder so the curl line reads as a template, not a fake request.
    if (param?.example !== undefined) return String(param.example);
    const schema = param?.schema;
    if (schema?.example !== undefined) return String(schema.example);
    if (Array.isArray(schema?.examples) && schema.examples.length > 0) {
      return String(schema.examples[0]);
    }
    return `{${name}}`;
  });
}

function buildQueryString(params: OASParameter[]): string {
  const required = params.filter((p) => p.in === 'query' && p.required);
  if (required.length === 0) return '';
  const parts = required.map((p) => `${p.name}=${encodeURIComponent(String(sampleParam(p)))}`);
  return `?${parts.join('&')}`;
}

function authHeaderFor(operation: OASOperation, doc: OASDocument): string | null {
  const security = operation.security ?? doc.security;
  if (!security?.length) return null;
  const schemes = doc.components?.securitySchemes ?? {};
  for (const req of security) {
    for (const name of Object.keys(req)) {
      const scheme = schemes[name];
      if (!scheme) continue;
      if (scheme.type === 'http' && scheme.scheme?.toLowerCase() === 'bearer') {
        return 'Authorization: Bearer YOUR_TOKEN';
      }
      if (scheme.type === 'http' && scheme.scheme?.toLowerCase() === 'basic') {
        return 'Authorization: Basic <base64-credentials>';
      }
      if (scheme.type === 'apiKey' && scheme.in === 'header' && scheme.name) {
        // Many APIs misuse `apiKey` with `name: Authorization`, expecting a
        // `Bearer ` prefix. If the header name is Authorization, default to
        // the Bearer convention; otherwise emit a raw value.
        if (scheme.name.toLowerCase() === 'authorization') {
          return 'Authorization: Bearer YOUR_TOKEN';
        }
        return `${scheme.name}: YOUR_API_KEY`;
      }
      if (scheme.type === 'apiKey') {
        // apiKey in query/cookie — fall back to Bearer; better than nothing.
        return 'Authorization: Bearer YOUR_TOKEN';
      }
    }
  }
  return 'Authorization: Bearer YOUR_TOKEN';
}

function sampleParam(param: OASParameter): unknown {
  if (param.example !== undefined) return param.example;
  return sampleFromSchema(param.schema, { paths: {} } as OASDocument) ?? `<${param.name}>`;
}

function sampleFromSchema(schema: OASSchema | undefined, doc: OASDocument, depth = 0): unknown {
  if (!schema || depth > 6) return null;
  if (schema.$ref) {
    const resolved = resolveSchemaRef(doc, schema.$ref);
    return resolved ? sampleFromSchema(resolved, doc, depth + 1) : null;
  }
  if (schema.example !== undefined) return schema.example;
  // OpenAPI 3.1+ uses `examples: [v1, v2, ...]` instead of single `example`.
  // The upgrader in fumadocs-openapi normalizes specs to 3.2.0, so prefer
  // this when present.
  if (Array.isArray(schema.examples) && schema.examples.length > 0) {
    return schema.examples[0];
  }
  if (schema.default !== undefined) return schema.default;
  if (schema.enum?.length) return schema.enum[0];

  const composite = schema.oneOf ?? schema.anyOf;
  if (composite?.length) return sampleFromSchema(composite[0], doc, depth + 1);
  if (schema.allOf?.length) {
    const merged: OASSchema = { type: 'object', properties: {}, required: [] };
    for (const part of schema.allOf) {
      const piece = part.$ref ? resolveSchemaRef(doc, part.$ref) : part;
      if (!piece) continue;
      Object.assign(merged.properties!, piece.properties);
      merged.required!.push(...(piece.required ?? []));
    }
    return sampleFromSchema(merged, doc, depth + 1);
  }

  const type = Array.isArray(schema.type) ? schema.type[0] : schema.type;
  if (type === 'object' || schema.properties) {
    const out: Record<string, unknown> = {};
    for (const [name, prop] of Object.entries(schema.properties ?? {})) {
      out[name] = sampleFromSchema(prop, doc, depth + 1);
    }
    return out;
  }
  if (type === 'array') {
    return [sampleFromSchema(schema.items, doc, depth + 1)];
  }
  if (type === 'string') {
    if (schema.format === 'date-time') return '2024-01-01T00:00:00Z';
    if (schema.format === 'date') return '2024-01-01';
    if (schema.format === 'uuid') return '00000000-0000-0000-0000-000000000000';
    if (schema.format === 'email') return 'user@example.com';
    if (schema.format === 'uri') return 'https://example.com';
    return 'string';
  }
  if (type === 'integer') return 0;
  if (type === 'number') return 0;
  if (type === 'boolean') return true;
  return null;
}

function resolveSchemaRef(doc: OASDocument, ref: string): OASSchema | undefined {
  if (!ref.startsWith('#/')) return undefined;
  const parts = ref.slice(2).split('/');
  let cur: unknown = doc;
  for (const p of parts) {
    if (cur === null || typeof cur !== 'object') return undefined;
    cur = (cur as Record<string, unknown>)[p];
  }
  return cur as OASSchema | undefined;
}

function escapeSingleQuotes(s: string): string {
  // Bash single-quoted strings can't contain single quotes; use the standard
  // close/open trick: 'foo'\''bar' → foo'bar
  return s.replace(/'/g, "'\\''");
}
