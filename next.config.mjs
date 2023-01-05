import nextMDX from "@next/mdx";
import { remarkPlugins } from "./mdx/remark.mjs";
import { rehypePlugins } from "./mdx/rehype.mjs";
import { recmaPlugins } from "./mdx/recma.mjs";

async function redirects() {
  return [
    {
      source: "/discord",
      destination: "https://discord.gg/EuesV2ZSnX",
      permanent: true,
    },
    {
      source: "/mailing-list",
      destination: "http://eepurl.com/hI3dCr",
      permanent: true,
    },
    // Legacy docs pages - These should be able to be removed after we
    // remove all legacy CLI + Cloud docs
    {
      source: "/docs/function-ide-guide",
      destination: "/docs/cloud/function-ide-guide",
      permanent: false,
    },
    {
      source: "/docs/using-the-inngest-cli",
      destination: "/docs/cloud/using-the-inngest-cli",
      permanent: false,
    },
  ];
}

const withMDX = nextMDX({
  options: {
    remarkPlugins,
    rehypePlugins,
    recmaPlugins,
  },
});

/** @type {import('next').NextConfig} */
const nextConfig = {
  redirects,
  reactStrictMode: true,
  pageExtensions: ["js", "jsx", "ts", "tsx", "mdx"],
  experimental: {
    scrollRestoration: true,
  },
};

export default withMDX(nextConfig);
