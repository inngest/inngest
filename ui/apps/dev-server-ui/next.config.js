// @ts-check

/** @type {import('next').NextConfig} */
const nextConfig = {
  distDir: './dist',
  transpilePackages: ['@inngest/components'],
};

module.exports = nextConfig;
