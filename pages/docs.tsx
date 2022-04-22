import React from "react";
import Head from "next/head";
import { useRouter } from "next/router";
import styled from "@emotion/styled";

import DocsNav from "../shared/Docs/DocsNav";
import Footer from "../shared/footer";
import { getAllDocs, Categories } from "../utils/docs";
import docsSyntaxHighlightingCSS from "../shared/Docs/docsSyntaxHighlighting";

export async function getStaticProps() {
  const categories = getAllDocs().categories;
  return { props: { categories: categories } };
}

export default function DocsHome(props) {
  return (
    <DocsLayout categories={props.categories}>
      <Head>
        <title>Inngest → Documentation & Guides</title>
      </Head>
      <Hero>
        <h1>Documentation</h1>
        {/* TODO: Quick start guide callouts, and graphic */}
      </Hero>

      <DocsContent>
        <div>
          <h2>What is Inngest</h2>
          <p>
            Inngest is a programmable event platform which allows you to
            aggregate every event in your business, and react to them by running
            code in real-time.
          </p>

          <p>
            We subscribe to every event in your stack, and allow you to run a
            DAG of serverless functions whenever specific events are received.
          </p>
          <p>
            Our platform allows you to build your product, ops, and internal
            flows behind a single abstraction: treating anything that happens
            across any service as a single event.
          </p>

          {/* Start building for free callout */}

          <h2>Discover Inngest</h2>

          <Discover>
            <div>
              <div>
                <h3>Getting Started</h3>
                <p>
                  A technical and non-technical introduction to the features of
                  Inngest, how it works, and step-by-step examples to get you
                  running in minutes.
                </p>
              </div>
              <ul>
                <li>
                  <a href="/docs/what-is-inngest">What is Inngest?</a>
                </li>
                <li>
                  <a href="/docs/how-inngest-works">How Inngest works</a>
                </li>
              </ul>
            </div>
          </Discover>
        </div>
      </DocsContent>
    </DocsLayout>
  );
}

export const DocsLayout: React.FC<{ categories: Categories }> = ({
  children,
  categories,
}) => {
  const router = useRouter();
  return (
    <>
      <DocsWrapper>
        <DocsNav categories={categories} />
        <Main>{children}</Main>
      </DocsWrapper>
      <Footer />
    </>
  );
};

// NOTE - We use em's here to base all sizing off our base 14px font-size in the docs
const DocsWrapper = styled.div`
  --border-color: var(--stroke-color);

  position: relative;
  display: grid;
  min-height: 100vh;
  grid-template-columns: 17em auto;
  gap: 4em;
  padding-right: 4em;
  font-size: 0.7rem;
  border-bottom: 1px solid var(--border-color);

  @media (max-width: 800px) {
    display: block;
    padding-right: 0;
  }
`;

const Main = styled.main`
  --docs-toc-width: 176px;

  display: grid;
  grid-template-columns: auto var(--docs-toc-width);
  gap: 4em;
  padding: 4em 0;

  // TOC changes to floating button - no need for grid
  @media (max-width: 1000px) {
    display: block;
    padding-top: 6em;
  }
  @media (max-width: 800px) {
    padding: 8em 2em 2em;
  }
`;

const Hero = styled.div`
  h1 {
    font-size: 2.5em;
    margin-bottom: 1em;
  }
`;

export const DocsContent = styled.article`
  --base-size: 16px;

  max-width: 720px;
  margin: 0 auto;
  padding-bottom: 10vh;
  font-size: 16px;

  h2,
  h3,
  h4,
  h5 {
    line-height: 1.5em;
  }
  h1 {
    font-size: 2em;
    margin-bottom: calc(2 * var(--base-size));
  }
  h2 {
    font-size: 1.5em;
    margin-top: calc(2 * var(--base-size));
  }
  h3 {
    font-size: 1.3em;
    margin-top: calc(1.5 * var(--base-size));
  }
  h3 {
    font-size: 1.3em;
  }
  p,
  ol,
  ul,
  li {
    font-size: 16px;
  }
  p {
    margin: 1em 0;
    line-height: 1.7em;
  }

  ol,
  ul {
    margin: 2em 0;
  }

  img {
    max-width: 100%;
    border-radius: var(--border-radius);
  }

  .featured-image {
    margin: 2em 0;
  }

  pre {
    margin: 1em 0;
    padding: 1rem;
    border-radius: 3px;
  }

  .language-id {
    display: none;
  }

  .tldr {
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius);
    padding: 3.5em 1.5em 1.5em;
    margin: 0 0 3em;
    position: relative;

    p,
    li {
      margin: 0;
    }
    p + p {
      margin: 1em 0 0;
    }

    ol,
    ul {
      margin: 1rem 0;
    }

    &:before {
      content: "TL;DR";
      display: block;
      position: absolute;
      top: 1.5em;
      left: 1.5em;
      opacity: 0.5;

      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 1px;
    }
  }

  // We export this to keep this file smaller and simpler
  ${docsSyntaxHighlightingCSS}
`;

const Discover = styled.div`
  > div {
    display: grid;
    grid-template-columns: 3fr 2fr;
    grid-gap: 2rem;
  }

  ul {
    margin-top: 3.5rem;
  }

  @media (max-width: 800px) {
    > div {
      grid-template-columns: 1fr;
      grid-gap: 0;
    }
    ul {
      margin-top: 1rem;
    }
  }
`;
