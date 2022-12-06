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
        source: "/mailing-list",
        destination: "http://eepurl.com/hI3dCr",
        permanent: true,
      },
      // Legacy docs pages - These should be able to be removed after we
      // remove all legacy CLI + Cloud docs
      {
        source: "/docs/quick-start",
        destination: "/docs",
        permanent: false,
      },
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
  },
};
