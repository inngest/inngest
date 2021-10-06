#!/usr/bin/env node

const fs = require('fs');
const library = require("./public/json/library.json");

const slugify = (str) => {
  return encodeURIComponent(str.toLowerCase().replace(" ", "-"));
}

const layout = fs.readFileSync("./layouts/library-item.js").toString();

library.forEach(item => {
  const title = slugify(item.title);
  fs.writeFileSync(`./pages/library/${title}.js`, layout.replace("$1", title));
});
