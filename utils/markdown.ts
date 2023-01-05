import { serialize } from "next-mdx-remote/serialize";
import rehypeSlug from "rehype-slug";
import rehypeRaw from "rehype-raw";
import rehypeAutolinkHeadings from "rehype-autolink-headings";

import { rehypeShiki } from "src/utils/code";
import { getHeadingsAsArray, Heading } from "src/utils/docs";

export type MDXFileMetadata = {
  slug: string;
  reading: {
    text: string;
    minutes: number;
    time: number;
    words: number;
  };
  [key: string]: any;
};

/**
 * A generic method to load and parse mdx files in a given directory
 * @param dir
 */
export async function loadMarkdownFilesMetadata<T>(
  dir: string
): Promise<T & MDXFileMetadata[]> {
  const fs = require("node:fs");
  const path = require("node:path");
  const matter = require("gray-matter");
  const readingTime = require("reading-time");

  const baseDir = path.join("./pages/", dir);

  // Iterate all files in the directory, then parse the markdown.
  const mdxFilenames = fs.readdirSync(baseDir);
  const filesMetadata = mdxFilenames.map((filename) => {
    const source = fs.readFileSync(path.join(baseDir, filename));

    const { data, content } = matter(source);
    data.reading = readingTime(content);
    data.slug = filename.replace(/.mdx?/, "");
    if (data.date) {
      data.humanDate = data.date.toLocaleDateString();
    }
    if (data.tags) {
      data.tags =
        typeof data.tags === "string"
          ? data.tags.split(",").map((t) => t.trim())
          : data.tags;
    }

    // Disregard the content as this is used for loading a list of files, e.g.
    // in a blog or careers page and just the frontmatter is used.
    // We need to stringify it since it wil be serialized at build-time.
    return data;
  });
  return filesMetadata;
}

export type MDXContent<T> = {
  content: string;
  headings: Heading[];
  compiledSource: string;
  metadata: T;
};

/**
 * A generic method to load and parse an mdx file
 * @param dir
 */
export async function loadMarkdownFile<T>(
  dir: string,
  slug: string
): Promise<MDXContent<T>> {
  const path = require("node:path");
  const fs = require("node:fs");
  const matter = require("gray-matter");
  const sourceFilename = path.join("./pages", dir, `${slug}.mdx`);
  const source = fs.readFileSync(sourceFilename, "utf8");
  const { content, data } = matter(source);
  const nodeTypes = [
    "mdxFlowExpression",
    "mdxJsxFlowElement",
    "mdxJsxTextElement",
    "mdxTextExpression",
    "mdxjsEsm",
  ];
  const serializedContent = await serialize(content, {
    // scope: { json: JSON.stringify(data) },
    mdxOptions: {
      remarkPlugins: [rehypeShiki],
      rehypePlugins: [
        [rehypeRaw, { passThrough: nodeTypes }],
        rehypeSlug,
        rehypeAutolinkHeadings,
      ],
    },
  });

  return {
    metadata: data,
    content,
    headings: getHeadingsAsArray(content),
    ...serializedContent,
  };
}
