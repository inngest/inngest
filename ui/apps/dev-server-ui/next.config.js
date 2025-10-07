// @ts-check

const CSP_HEADER = `
  default-src 'self';
`.replace(/\n/g, '');

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
            key: 'Content-Security-Policy',
            value: CSP_HEADER,
          },
        ],
      },
    ];
  },
};

module.exports = nextConfig;
