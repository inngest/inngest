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
    ];
  },
};
