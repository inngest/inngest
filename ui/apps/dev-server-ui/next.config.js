// @ts-check

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  distDir: './dist',
  transpilePackages: ['@inngest/components'],
};

module.exports = nextConfig;
