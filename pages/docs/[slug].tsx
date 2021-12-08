import styled from "@emotion/styled";
import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import rehypeSlug from "rehype-slug";
import { getAllDocs, getDocs, DocScope } from "../../utils/docs";
import { DocsLayout, DocsContent } from "../docs";

export default function DocLayout(props: any) {
  // TODO: Add previous / next.  Add layout.
  const scope: DocScope = JSON.parse(props.post.scope.json);

  return (
    <DocsLayout>
      <DocsContent>
        <div>
          <h2>{scope.title}</h2>

          <h5>On this page</h5>
          <TOC>
            {Object.values(scope.toc).sort((a, b) => a.order - b.order).map(h => {
            return (
              <li><a href={`#${h.slug}`}>{h.title}</a></li>
            );
            })}
          </TOC>

          <MDXRemote compiledSource={props.post.compiledSource} scope={scope} />
        </div>
        </DocsContent>
    </DocsLayout>
  );
}


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
    scope: { json: JSON.stringify(scope) },
    mdxOptions: {
      rehypePlugins: [rehypeSlug],
    },
  });
  return { props: { post } };
}

const TOC = styled.ol`
  margin: 0 0 4rem 1rem;
  padding: 0;
  font-size: 12px;

  li a {
    display: block;
    padding: .5rem 0;
    text-decoration: none;

  }
  li {
    border-bottom: 1px solid #ffffff19;
  }

`;
