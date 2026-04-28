#!/usr/bin/env node
/**
 * Post-build script that produces a Vercel Build Output API v3 structure
 * for TanStack Start v1's SSR output.
 *
 * TanStack Start outputs:
 *   dist/client/  — static assets (JS, CSS, api-specs)
 *   dist/server/  — SSR handler (server.js + assets/) that exports server.fetch()
 *
 * This script:
 *   1. Copies dist/client/ → .vercel/output/static/
 *   2. Bundles dist/server/server.js (+ all its node_modules deps) into a single
 *      server-bundle.mjs using Vite's programmatic build API, to stay well under
 *      Vercel's 15k-file upload limit.
 *   3. Wraps server-bundle.mjs in a Node.js req/res → Web Fetch API adapter.
 *   4. Writes .vercel/output/config.json routing.
 */
import { cpSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = fileURLToPath(new URL('..', import.meta.url));
const VERCEL_OUT = join(ROOT, '.vercel/output');
const STATIC_OUT = join(VERCEL_OUT, 'static');
const FUNC_DIR = join(VERCEL_OUT, 'functions/ssr.func');

// ── 1. Static assets ─────────────────────────────────────────────────────────
console.log('[vercel] Copying static assets…');
rmSync(STATIC_OUT, { recursive: true, force: true });
cpSync(join(ROOT, 'dist/client'), STATIC_OUT, { recursive: true });

// ── 2. Bundle SSR server into a single file ───────────────────────────────────
// dist/server/server.js imports from bare node_modules specifiers (react,
// @tanstack/react-router, etc.) and dynamically imports its own asset chunks
// (router, source, search, etc.).  Bundling everything into one self-contained
// .mjs avoids having thousands of node_modules files in the function directory.
console.log('[vercel] Bundling SSR server (this may take a moment)…');

// Vite is already installed — use its programmatic build API.
// Dynamic import is required because this script is ESM but we can't use
// top-level vite imports without a vite.config in scope.
const { build } = await import('vite');

rmSync(join(VERCEL_OUT, 'functions'), { recursive: true, force: true });
mkdirSync(FUNC_DIR, { recursive: true });

await build({
  root: ROOT,
  configFile: false,
  logLevel: 'warn',
  // noExternal: true bundles all node_modules into the output instead of
  // leaving them as external bare-specifier imports.
  ssr: { noExternal: true },
  build: {
    ssr: true,
    outDir: FUNC_DIR,
    emptyOutDir: false,
    rollupOptions: {
      input: resolve(ROOT, 'dist/server/server.js'),
      // Only node: built-ins remain external — they're always available in the
      // Vercel Node.js runtime without needing node_modules.
      external: [/^node:/],
      output: {
        format: 'esm',
        entryFileNames: 'server-bundle.mjs',
        // Inline all dynamic imports (route chunks, MDX assets, etc.) so the
        // function directory stays to a handful of files.
        inlineDynamicImports: true,
      },
    },
  },
});

// Vite copies public/ assets into outDir (api-specs JSON/YAML).  Those belong
// in static, not in the function — remove them from the function directory.
rmSync(join(FUNC_DIR, 'api-specs'), { recursive: true, force: true });

// ── 3. Node.js req/res → Web Fetch API adapter ───────────────────────────────
writeFileSync(
  join(FUNC_DIR, 'handler.mjs'),
  `import server from './server-bundle.mjs';

export default async function handler(req, res) {
  // Reconstruct a full URL (Vercel sets x-forwarded-proto / x-forwarded-host)
  const proto = req.headers['x-forwarded-proto'] ?? 'https';
  const host = req.headers['x-forwarded-host'] ?? req.headers['host'];
  const url = \`\${proto}://\${host}\${req.url}\`;

  const headers = new Headers();
  for (const [k, v] of Object.entries(req.headers)) {
    if (v != null) headers.set(k, Array.isArray(v) ? v.join(', ') : v);
  }

  let body = undefined;
  if (req.method !== 'GET' && req.method !== 'HEAD') {
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    body = Buffer.concat(chunks);
  }

  const request = new Request(url, { method: req.method, headers, body });
  const response = await server.fetch(request);

  res.statusCode = response.status;
  for (const [k, v] of response.headers.entries()) {
    res.setHeader(k, v);
  }

  if (response.body) {
    const reader = response.body.getReader();
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        res.write(value);
      }
    } finally {
      res.end();
    }
  } else {
    res.end();
  }
}
`
);

// .vc-config.json tells Vercel how to invoke the function
writeFileSync(
  join(FUNC_DIR, '.vc-config.json'),
  JSON.stringify(
    {
      runtime: 'nodejs22.x',
      handler: 'handler.mjs',
      launchAt: 'request',
      supportsResponseStreaming: true,
    },
    null,
    2
  )
);

// ── 4. Routing ────────────────────────────────────────────────────────────────
// `handle: filesystem` serves exact matches from .vercel/output/static/
// (JS chunks, CSS, api-specs, etc.) before falling through to the SSR function.
writeFileSync(
  join(VERCEL_OUT, 'config.json'),
  JSON.stringify(
    {
      version: 3,
      routes: [{ handle: 'filesystem' }, { src: '/.*', dest: '/ssr' }],
    },
    null,
    2
  )
);

console.log('[vercel] Build output ready at .vercel/output/');
