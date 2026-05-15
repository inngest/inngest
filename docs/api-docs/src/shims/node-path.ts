// Minimal browser-compatible shim for node:path.
// fumadocs-mdx/runtime/server imports node:path at the top level, which esbuild
// evaluates during dev-mode pre-bundling. This shim prevents the crash so that
// the server runtime can initialize in the browser (SPA mode). Rollup tree-shakes
// the import away in production so the shim is only active in dev.

export function join(...parts: string[]): string {
  return parts.filter(Boolean).join('/').replace(/\\/g, '/').replace(/\/+/g, '/');
}

export function dirname(p: string): string {
  const dir = p.replace(/\\/g, '/').replace(/\/[^/]*$/, '');
  return dir || (p.startsWith('/') ? '/' : '.');
}

export function basename(p: string, ext?: string): string {
  const base = p.replace(/\\/g, '/').split('/').pop() ?? '';
  return ext && base.endsWith(ext) ? base.slice(0, -ext.length) : base;
}

export default { join, dirname, basename };
