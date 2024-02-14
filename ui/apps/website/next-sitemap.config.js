/** @type {import('next-sitemap').IConfig} */
module.exports = {
  siteUrl: "https://www.inngest.com",
  generateRobotsTxt: true,
  // Don't index the _underscore prefix directories
  exclude: ["*/_*"],
};
