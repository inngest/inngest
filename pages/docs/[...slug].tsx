import { useEffect } from "react";
import Head from "next/head";
import styled from "@emotion/styled";
import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import rehypeSlug from "rehype-slug";
// local
import { getAllDocs, getDocs, DocScope, Categories } from "../../utils/docs";
import { DocsLayout, DocsContent, InnerDocsContent } from "../docs";
import { highlight } from "../../utils/code";

export default function DocLayout(props: any) {
  const scope: DocScope & { categories: Categories } = JSON.parse(
    props.post.scope.json
  );

  useEffect(() => {
    const cb = (entries: IntersectionObserverEntry[]) => {
      if (entries.length === 0) {
        return;
      }
      document
        .querySelector("#toc-side")
        ?.classList?.toggle("visible", !entries.shift().isIntersecting);
    };
    const observer = new IntersectionObserver(cb, {});
    observer.observe(document.querySelector("#toc"));
    return () => observer.disconnect();
  }, []);

  return (
    <DocsLayout categories={scope.categories}>
      <Head>
        <title>{scope.title} → Inngest docs</title>
      </Head>
      <DocsContent>
        <div>
          <h2>{scope.title}</h2>

          <h5>On this page</h5>
          <TOC id="toc">
            {Object.values(scope.toc)
              .sort((a, b) => a.order - b.order)
              .map((h, n) => {
                return (
                  <li key={h.slug}>
                    <a href={`#${h.slug}`}>
                      {n + 1}. {h.title}
                    </a>
                  </li>
                );
              })}
          </TOC>

          <InnerDocsContent>
            <MDXRemote
              compiledSource={props.post.compiledSource}
              scope={scope}
            />
            {/* TODO: Add a prev / next button */}
          </InnerDocsContent>
        </div>
        <div>
          <TOCSide id="toc-side">
            <h5>On this page</h5>

            {Object.values(scope.toc)
              .sort((a, b) => a.order - b.order)
              .map((h, n) => {
                return (
                  <li key={h.slug}>
                    <a href={`#${h.slug}`}>
                      {n + 1}. {h.title}
                    </a>
                  </li>
                );
              })}
          </TOCSide>
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
      remarkPlugins: [highlight],
      rehypePlugins: [rehypeSlug],
    },
  });
  return { props: { post } };
}

const TOC = styled.ol`
  margin: 1.5rem 0 4rem;
  padding: 0;
  font-size: 0.9rem;

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

const TOCSide = styled.ol`
  position: fixed;
  top: calc(70px + 5vh);
  opacity: 0;
  pointer-events: none;
  transition: all 0.3s;
  font-size: 0.9rem;

  z-index: 0;

  &.visible {
    display: block;
    opacity: 1;
    pointer-events: auto;
  }

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

  @media (max-width: 800px) {
    &.visible {
      opacity: 0;
      display: none;
    }
  }
`;
