import { useState } from "react";
import Head from "next/head";
import styled from "@emotion/styled";
import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import rehypeSlug from "rehype-slug";
// local
import {
  getAllDocs,
  getDocs,
  DocScope,
  Categories,
  Headings,
} from "../../utils/docs";
import { DocsLayout, DocsContent, InnerDocsContent } from "../docs";
import { highlight } from "../../utils/code";

export default function DocLayout(props: any) {
  const scope: DocScope & { categories: Categories } = JSON.parse(
    props.post.scope.json
  );

  return (
    <DocsLayout categories={scope.categories}>
      <Head>
        <title>{scope.title} → Inngest docs</title>
      </Head>
      <DocsContent>
        <h1>{scope.title}</h1>

        <InnerDocsContent>
          <MDXRemote compiledSource={props.post.compiledSource} scope={scope} />
          {/* TODO: Add a prev / next button */}
        </InnerDocsContent>
        <TOCSide toc={scope.toc} />
      </DocsContent>
    </DocsLayout>
  );
}

const TOCSide: React.FC<{ toc: Headings }> = ({ toc = {} }) => {
  const [isExpanded, setExpanded] = useState(false);
  return (
    <TOC isExpanded={isExpanded}>
      <h5 onClick={() => setExpanded(!isExpanded)}>On this page</h5>
      <ol>
        {Object.values(toc)
          .sort((a, b) => a.order - b.order)
          .map((h, n) => {
            return (
              <li key={h.slug}>
                <a href={`#${h.slug}`} onClick={() => setExpanded(!isExpanded)}>
                  {h.title}
                </a>
              </li>
            );
          })}
      </ol>
    </TOC>
  );
};

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
      remarkPlugins: [highlight],
      rehypePlugins: [rehypeSlug],
    },
  });
  return { props: { post, htmlClassName: "docs" } };
}

const TOC = styled.nav<{ isExpanded: boolean }>`
  position: fixed;
  top: 3em;
  right: 2em;
  max-width: var(--docs-toc-width);
  pointer-events: auto;
  transition: all 0.3s;
  font-size: 14px;
  text-align: right;

  z-index: 0;

  h5,
  ol,
  li {
    margin: 1em 0;
    font-size: 1em;
  }

  a {
    display: block;
    text-decoration: none;
    color: var(--font-color-secondary);
    &:hover {
      color: var(--font-color-primary);
    }
  }

  ol {
    padding: 0;
  }
  li {
    list-style: none;
  }

  // See relatd MQs in docs.tsx
  @media (max-width: 980px) {
    padding: 0 1.5em;
    max-width: calc(var(--docs-toc-width) + 3em);
    background-color: var(--bg-color);
    border-radius: var(--border-radius);
    box-shadow: ${({ isExpanded }) =>
      isExpanded ? "var(--box-shadow)" : "none"};
    ol {
      display: ${({ isExpanded }) => (isExpanded ? "block" : "none")};
    }
  }
  @media (max-width: 800px) {
    display: none;
  }
`;
