import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import rehypeSlug from "rehype-slug";

import { getAllDocs, getDocs } from "../../utils/docs";

export default function DocLayout(props) {
  // TODO: Add previous / next.  Add layout.
  const scope = JSON.parse(props.post.scope);

  return (
    <div>
      <h1>docs</h1>
      <pre>{JSON.stringify(scope, undefined, "  ")}</pre>
      <MDXRemote compiledSource={props.post.compiledSource} scope={scope} />
    </div>
  );
}

// const docs = require('../../utils/docs');

// This function gets called at build time to figure out which URLs
// we need to statically compile.
//
// These URLs will be treated as individual pages. getStaticProps is
// called for each URL with the slug in params.
export async function getStaticPaths() {
  const docs = getAllDocs();
  return { paths: docs.slugs, fallback: false };
}

// This function also gets called at build time to generate specific content.
export async function getStaticProps({ params }) {
  // These are required here as this function is not included in frontend
  // browser builds.
  const slug = params.slug; //.replace(/^\/docs\//, "");

  const docs = getDocs("/docs/" + slug);
  if (!docs) {
    throw new Error("unable to find docs for " + slug);
  }
  const { content, scope } = docs;

  const post = await serialize(content, {
    scope: JSON.stringify(scope),
    mdxOptions: {
      rehypePlugins: [rehypeSlug],
    },
  });

  return { props: { post } };
}
