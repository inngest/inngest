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
  Doc,
  Docs,
} from "../../utils/docs";
import Code from "src/shared/Code";
import { DocsLayout, DocsContent } from "../docs";
import { highlight } from "../../utils/code";

export default function DocLayout(props: any) {
  const scope: DocScope = JSON.parse(props.post.scope.json);
  const sections = JSON.parse(props.sections);

  const formattedTitle = scope.title.replace(/`(.+)`/, "<code>$1</code>");

  // metadata
  const titleWithCategory =
    scope.category !== scope.title
      ? `${scope.category}: ${scope.title}`
      : scope.title;
  const metaTitle = `${titleWithCategory} - Inngest Documentation`;
  const description =
    scope.description || `Inngest documentation for ${scope.title}`;
  // Social preview images are rendered at build time via `yarn render-social-preview-images`, also see Makefile
  // but are rendered dynamically here to allow for previews
  const flattenedSlug = scope.slug.replace(/\//, "--");
  const generatedPreviewImage =
    process.env.NODE_ENV === "development"
      ? `${process.env.NEXT_PUBLIC_HOST}/api/socialPreviewImage?title=${titleWithCategory}`
      : `${process.env.NEXT_PUBLIC_HOST}/assets/social-previews/${flattenedSlug}.png`;
  const socialImage = scope.image
    ? `${process.env.NEXT_PUBLIC_HOST}${scope.image}`
    : generatedPreviewImage;

  return (
    <DocsLayout sections={sections}>
      <Head>
        <title>{metaTitle}</title>
        <meta name="description" content={description}></meta>
        <meta property="og:title" content={metaTitle} />
        <meta property="og:description" content={description} />
        <meta property="og:type" content="article" />
        <meta
          property="og:url"
          content={`${process.env.NEXT_PUBLIC_HOST}/docs/${scope.slug}`}
        />
        <meta property="og:image" content={socialImage} />
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:site" content="@inngest" />
        <meta name="twitter:title" content={metaTitle} />
        <meta name="twitter:image" content={socialImage} />
        {scope.hide ? <meta name="robots" content="noindex,nofollow" /> : null}
      </Head>
      <DocsContent hasTOC={true}>
        <header>
          <h1
            dangerouslySetInnerHTML={{
              __html: formattedTitle,
            }}
          ></h1>
          {!!scope.image && (
            <img
              src={scope.image}
              className="featured-image"
              alt="Featured image"
            />
          )}
        </header>
        <MDXRemote
          compiledSource={props.post.compiledSource}
          scope={scope}
          components={{ Code: Code }}
        />

        <FooterLinks className="grid sm:grid-cols-1 md:grid-cols-2 gap-4 py-8">
          <div>
            {props.prev && (
              <a
                href={`/docs/${props.prev.slug}`}
                className="shadow-md rounded-sm p-4 border-slate-200 border-2 color-inherit hover:shadow-2xl bg-white"
              >
                <small>Previous</small>
                {props.prev.scope.title}
              </a>
            )}
          </div>
          <div>
            {props.next && (
              <a
                href={`/docs/${props.next.slug}`}
                className="shadow-md rounded-sm p-4 border-slate-200 border-2 color-inherit hover:shadow-2xl bg-white text-right"
              >
                <small>Next</small>
                {props.next.scope.title}
              </a>
            )}
          </div>
        </FooterLinks>
      </DocsContent>
      <aside>
        <TOCSide toc={scope.toc} />
      </aside>
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
  const parsedSlug = Array.isArray(slug) ? slug.join("/") : slug;
  const docs = getDocs(parsedSlug);
  const all = getAllDocs();

  // Find next and previous articles based off of slug order.
  const [next, prev] = (() => {
    const idx = all.slugs.findIndex((s) => s === "/docs/" + parsedSlug);
    // Remove '/docs/' from slugs.
    const subSlugs = all.slugs.map((slug) => slug.substring(6));
    // console.log(subSlugs, idx, "/docs/" + parsedSlug);

    const next: Doc | null =
      all.docs[
        subSlugs
          .slice(idx + 1)
          .find((slug) => all.docs[slug] && !all.docs[slug].scope.hide)
      ] || null;

    const prev: Doc | null =
      (idx > 0 &&
        all.docs[
          subSlugs
            .reverse()
            .slice(-idx)
            .find((slug) => all.docs[slug] && !all.docs[slug].scope.hide)
        ]) ||
      null;

    return [next, prev];
  })();

  if (!docs) {
    throw new Error("unable to find docs for " + parsedSlug);
  }

  const { content } = docs;

  // Add categories to the scope such that we can show them in the UI.
  const scope = { ...docs.scope };

  const post = await serialize(content, {
    scope: { json: JSON.stringify(scope) },
    mdxOptions: {
      remarkPlugins: [highlight],
      rehypePlugins: [rehypeSlug],
    },
  });

  return {
    props: {
      post,
      sections: JSON.stringify(all.sections),
      next,
      prev,
      htmlClassName: "docs",
      meta: { disabled: true },
    },
  };
}

const TOC = styled.nav<{ isExpanded: boolean }>`
  position: sticky;
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

  ol,
  ol ul {
    padding: 0;
    margin: 0;
  }
  li {
    list-style: none;
  }

  // See relatd MQs in docs.tsx
  @media (max-width: 1000px) {
    position: fixed;
    top: 3em;
    right: 1.5em;
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

const FooterLinks = styled.div`
  a {
    display: block;
    small {
      display: block;
    }
  }
`;
