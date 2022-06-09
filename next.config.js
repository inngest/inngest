module.exports = {
  // All redirects must also be copied to ./_redirects for production on Cloudflare Pages
  async redirects() {
    return [
      {
        source: "/discord",
        destination: "https://discord.gg/EuesV2ZSnX",
        permanent: true,
      },
      {
        source: "/docs",
        destination: "/docs/what-is-inngest",
        permanent: false,
      },
    ];
  },
};
