import styled from "@emotion/styled";
import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import rehypeSlug from "rehype-slug";
import { getAllDocs, getDocs, DocScope, Categories } from "../../utils/docs";
import { DocsLayout, DocsContent } from "../docs";

export default function DocLayout(props: any) {
  // TODO: Add previous / next.  Add layout.
  const scope: DocScope & { categories: Categories } = JSON.parse(
    props.post.scope.json
  );

  console.log(scope);

  return (
    <DocsLayout categories={scope.categories}>
      <DocsContent>
        <div>
          <h2>{scope.title}</h2>

          <h5>On this page</h5>
          <TOC>
            {Object.values(scope.toc)
              .sort((a, b) => a.order - b.order)
              .map((h, n) => {
                return (
                  <li>
                    <a href={`#${h.slug}`}>
                      {n + 1}. {h.title}
                    </a>
                  </li>
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
  const slug = params.slug;

  const categories = getAllDocs().categories;
  const docs = getDocs(Array.isArray(slug) ? slug.join("/") : slug);

  if (!docs) {
    throw new Error("unable to find docs for " + slug);
  }

  const { content } = docs;

  // Add categories to the scope such that we can show them in the UI.
  const scope = { ...docs.scope, categories };

  const post = await serialize(content, {
    scope: { json: JSON.stringify(scope) },
    mdxOptions: {
      rehypePlugins: [rehypeSlug],
    },
  });
  return { props: { post } };
}

const TOC = styled.ol`
  margin: 0 0 4rem;
  padding: 0;
  font-size: 12px;

  li a {
    display: block;
    padding: 0.5rem 0;
    text-decoration: none;
    opacity: 0.85;
  }
  li {
    border-bottom: 1px solid #ffffff19;
    transition: all 0.3s;
    text-indent: 0;
    list-style: none;
    &:hover {
      padding-left: 0.5rem;
    }
  }
`;
