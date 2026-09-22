import { cpSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { resolve } from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const output = resolve(root, '.vercel/output');
// the website forwards /dev paths unchanged.  files in static/dev keep
// those paths valid for documents and assets.
const publicDir = resolve(output, 'static/dev');
mkdirSync(publicDir, { recursive: true });
cpSync(resolve(root, 'dist/client'), publicDir, { recursive: true });

// src/routeTree.gen.ts lists the dashboard routes.  deployment rules must
// include them so direct visits and page refreshes load the UI.
const generated = readFileSync(resolve(root, 'src/routeTree.gen.ts'), 'utf8');
const routeTypes = generated.match(
  /export interface FileRoutesByTo \{([\s\S]*?)\n\}/,
)?.[1];
if (!routeTypes)
  throw new Error('Cannot read dashboard routes from routeTree.gen.ts');
const routes = [...routeTypes.matchAll(/'([^']+)':/g)].map((match) => match[1]);
// dynamic route parameters need explicit matching rules.  literal parameter
// names in config.json do not match URLs that contain actual values.
if (routes.some((route) => !/^\/[a-z/-]*$/.test(route))) {
  throw new Error(
    'Hosted deployment needs an explicit rule for new dynamic routes',
  );
}

writeFileSync(
  resolve(output, 'config.json'),
  JSON.stringify(
    {
      version: 3,
      routes: [
        {
          src: '/dev(/.*)?',
          headers: {
            'Permissions-Policy':
              'loopback-network=(self), local-network-access=(self)',
          },
          continue: true,
        },
        // /dev/index.html contains the public setup instructions.  only this
        // entry page is indexed because dashboard pages depend on localhost.
        {
          src: '/dev',
          dest: '/dev/index.html',
          headers: { 'X-Robots-Tag': 'index, follow' },
        },
        { src: '/dev/', status: 308, headers: { Location: '/dev' } },
        {
          src: '/dev/.+',
          headers: { 'X-Robots-Tag': 'noindex, follow' },
          continue: true,
        },
        // only known dashboard paths use /dev/_shell.html.  unknown paths and
        // missing assets reach the 404 rule instead of returning dashboard HTML.
        ...routes
          .filter((route) => route !== '/')
          .map((route) => ({
            src: `/dev${route}/?`,
            dest: '/dev/_shell.html',
          })),
        { handle: 'filesystem' },
        { src: '/.*', status: 404, dest: '/dev/404.html' },
      ],
    },
    null,
    2,
  ),
);

writeFileSync(
  resolve(publicDir, '404.html'),
  '<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="robots" content="noindex"><title>Page not found | Inngest</title></head><body><main><h1>Page not found</h1><p><a href="/dev">Return to the Dev Server</a></p></main></body></html>',
);
