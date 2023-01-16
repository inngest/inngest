import nextMDX from "@next/mdx";
import { remarkPlugins } from "./mdx/remark.mjs";
import { rehypePlugins } from "./mdx/rehype.mjs";
import { recmaPlugins } from "./mdx/recma.mjs";

// All permanent redirects (source -> destination)
const legacyDocsUrls = [
  ["/docs/what-is-inngest", "/docs"],
  ["/docs/reference/functions/retries", "/docs/functions/retries"],
  ["/docs/creating-an-event-key", "/docs/events/creating-an-event-key"],
  ["/docs/writing-and-running-fuctions", "/docs/functions"], //typo
  ["/docs/cli/steps/", "/docs/functions/multi-step"],
  ["/docs/local-development", "/docs/functions/testing-functions"],
  ["/docs/events/sources/sdks", "/docs/events"],
  ["/docs/deploying-fuctions", "/docs/deploy"],
  ["/docs/how-inngest-works", "/docs"], // TODO/DOCS redirect this to new concepts page
];

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
    {
      // From the UI's source editing page:
      source: "/docs/event-webhooks",
      destination: "/docs/events/webhooks",
      permanent: true,
    },
    ...legacyDocsUrls.map(([source, destination]) => ({
      source,
      destination,
      permanent: true,
    })),
    {
      source: "/library/:path*",
      destination: "/patterns",
      permanent: true,
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
