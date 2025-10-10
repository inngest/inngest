// @ts-check

const INNGEST_FONT_CDN_URL = 'https://fonts-cdn.inngest.com';

const MONACO_EDITOR_CDN_URL = 'https://cdn.jsdelivr.net/npm/monaco-editor@0.43.0/min/vs';
const MONACO_EDITOR_CDN_SCRIPT_URLS = [
  `${MONACO_EDITOR_CDN_URL}/base/common/worker/simpleWorker.nls.js`,
  `${MONACO_EDITOR_CDN_URL}/base/worker/workerMain.js`,
  `${MONACO_EDITOR_CDN_URL}/language/json/jsonMode.js`,
  `${MONACO_EDITOR_CDN_URL}/language/json/jsonWorker.js`,
  `${MONACO_EDITOR_CDN_URL}/editor/editor.main.js`,
  `${MONACO_EDITOR_CDN_URL}/editor/editor.main.nls.js`,
  `${MONACO_EDITOR_CDN_URL}/loader.js`,
];
const MONACO_EDITOR_CDN_FONT_URL = `${MONACO_EDITOR_CDN_URL}/base/browser/ui/codicons/codicon/codicon.ttf`;
const MONACO_EDITOR_CDN_STYLE_URL = `${MONACO_EDITOR_CDN_URL}/editor/editor.main.css`;

const CSP_HEADER = `
 base-uri 'self';
 connect-src 'self' http://localhost:8288;
 default-src 'self';
 font-src 'self' ${INNGEST_FONT_CDN_URL} ${MONACO_EDITOR_CDN_FONT_URL};
 script-src 'self' ${MONACO_EDITOR_CDN_SCRIPT_URLS.join(' ')} 'unsafe-eval' 'unsafe-inline';
 style-src 'self' ${MONACO_EDITOR_CDN_STYLE_URL} 'unsafe-inline';
 worker-src 'self' blob:;
`.replace(/\n/g, '');

console.log(CSP_HEADER);

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  distDir: './dist',
  transpilePackages: ['@inngest/components'],
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'Content-Security-Policy-Report-Only',
            value: CSP_HEADER,
          },
        ],
      },
    ];
  },
};

module.exports = nextConfig;
