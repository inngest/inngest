#!/usr/bin/env node

// This provides a framework for constructing blog posts from markdown.  It's relatively
// small, and creates nextjs pages for each .md post, stripping the JSON data at the top.

const fs = require('fs');
const marked = require('marked');
const { exec } = require('child_process');

const path = "./pages/_blogposts/";

const layout = fs.readFileSync(path + "layout.js").toString();

const run = () => {
  // Remove all old content in case of renames
  fs.rmSync("./pages/blog", { recursive: true });
  fs.mkdirSync("./pages/blog");

  // Create each blog post.
  fs.readdirSync(path).forEach(fname => {
    const contents = fs.readFileSync(path + fname);
    process(contents.toString());
  });
};

const process = (contents) => {
  const [m, post] = contents.split("~~~");

  if (!post) return

  const parsed = marked(post);
  const meta = JSON.parse(m);

  // We always add the date below the H1.
  let html = parsed.replace("</h1>", `</h1><div className='blog--date'>${meta.date}</div>`);
  html = layout.replace("$1", html);

  fs.writeFileSync(`./pages/blog/${meta.slug}.js`, html)

  // TODO: Add redirects, add next/previous posts, add title
}

run();
exec('yarn prettier');
