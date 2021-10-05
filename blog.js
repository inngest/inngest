#!/usr/bin/env node
// This provides a framework for constructing blog posts from markdown.  It's relatively
// small, and creates nextjs pages for each .md post, stripping the JSON data at the top.

const fs = require('fs');
const marked = require('marked');
const { exec } = require('child_process');

const path = "./_blogposts/";

const layout = fs.readFileSync(path + "layout.js").toString();
const index = fs.readFileSync(path + "blog.js").toString();

const run = () => {
  // Remove all old content in case of renames
  fs.rmSync("./pages/blog", { recursive: true });
  fs.mkdirSync("./pages/blog");

  // Posts is an array of objects, containing the metadata + content.
  const posts = fs.readdirSync(path).map(fname => {
    const contents = fs.readFileSync(path + fname);
    return parse(contents.toString());
  }).filter(Boolean);

  posts.sort((a, b) => { a.date.localeCompare(b.date) });

  // Create each blog post
  posts.forEach(process);

  // Create the blog homepage.
  createList(posts);
};

const parse = (contents) => {
  let [meta, callout, markdown] = contents.split("~~~");

  // Sometimes there might be no callout.
  if (callout && !markdown) {
    markdown = callout;
    callout = "";
  }

  if (!meta || !markdown) return

  const post = JSON.parse(meta);

  post.content = marked((markdown || "").trim());
  post.callout = marked((callout || "").trim());

  return post;
}

const process = (post) => {
  if (!post) return
  // We always add the date below the H1.
  let html = `<h1>${post.heading}</h1><div className='blog--date'>${post.date}</div>`;

  if (post.callout) {
    html += `<div className='blog--callout'>${post.callout}</div>`;
  }

  html +=  post.content;
  html = layout.replace("$1", html);

  fs.writeFileSync(`./pages/blog/${post.slug}.js`, html)

  // TODO: Add redirects, add next/previous posts, add title
}

const createList = (posts) => {
  let html = posts.map(post => {
    return `
      <a href="/blog/${post.slug}" className='post--item'>
        <h2>${post.heading}</h2>
        ${post.subtitle ? `<p>${post.subtitle}</p>` : ""}
      </a>
      `;
  });

  html = index.replace("$1", html.join(""));
  fs.writeFileSync(`./pages/blog.js`, html);
}

run();
exec('yarn prettier');
