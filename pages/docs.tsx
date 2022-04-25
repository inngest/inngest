import React from "react";
import Head from "next/head";
import { useRouter } from "next/router";
import styled from "@emotion/styled";

import DocsNav from "../shared/Docs/DocsNav";
import Footer from "../shared/footer";
import Button from "../shared/Button";
import ArrowUpRightIcon from "../shared/Icons/ArrowUpRight";
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
      <DocsContent>
        <Hero>
          <h1>Introduction to Inngest</h1>
        </Hero>

        <p>
          The Inngest platform allows you to easily build, deploy, and manage
          <strong>serverless functions</strong> that run whenever events happen,
          from any source (SDKs, APIs, webhooks, or pub/sub).
          {/* TODO - Link to future docs */}
        </p>
        <p>
          Inngest's tooling makes developers lives easier: get auto-typed
          events, develop & test serverless functions locally, backtest using
          historical data, release with full version histories, apply immediate
          rollbacks, and get full observability, metrics, monitoring, and audit
          trails out of the box. Inngest's serverless platform allows anyone to
          build event-driven software that scales.
        </p>

        <h2>What is Inngest?</h2>
        <p>
          Inngest is an <strong>event-driven serverless platform</strong> that
          lets you <strong>focus on your product</strong> by giving you all the
          tools you need to build, test, and ship reactive serverless functions
          faster than ever before.
        </p>
        <p>
          We've taken our experience building systems for event-driven
          architecture and async workloads and created a platform that allows
          you to run any serverless function that reacts to events. Inngest can
          be used in place of an event bus or message queue as well as the
          runtime for your event consumers or workers.
        </p>
        <p>
          Inngest allows you to build complex applications without all of the
          boilerplate and endless infrastructure configuration. No need to waste
          time writing boilerplate consumer code or polling logic. Don't waste
          time configuring retry policies, dead letter queues, or other
          infrastructure.
        </p>

        <p>
          Leave the boring stuff to us and focus on shipping amazing products,
          getting your idea to market faster than ever before.
        </p>

        <p>
          Check out our quick start guides below or read more about{" "}
          <a href="/docs/high-level-architecture">
            the system architecture here
          </a>
          .
        </p>

        <h2>Quick start guides</h2>

        <FeaturedDocs>
          <FeaturedDoc>
            <h3>Learn our cli in 2 minutes</h3>
            <p>
              Use the Inngest CLI to build, test, and deploy functions in any
              language.
            </p>

            <Button kind="primary" size="small" href="/docs/cli">
              <ArrowUpRightIcon /> Read the guide
            </Button>
          </FeaturedDoc>

          <FeaturedDoc>
            <h3>Use our browser IDE</h3>
            <p>
              Create, test and deploy functions right from your browser. No
              install needed.
            </p>

            <Button kind="primary" size="small" href="/docs/cli">
              <ArrowUpRightIcon /> Read the guide
            </Button>
          </FeaturedDoc>
        </FeaturedDocs>

        <h2>Learn the key concepts</h2>

        <ul>
          <li>
            <a href="/docs/quick-start">Quick start</a>
          </li>
          <li>
            <a href="/docs/event-format-and-structure">Event format</a>
          </li>
          <li>
            <a href="/docs/event-user-audit-trails">Event user attribution</a>
          </li>
        </ul>

        {/* Start building for free callout */}
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
    margin: 1.5em 0;
  }

  aside,
  video {
    margin: 1em 0;
  }

  aside {
    padding: 1.4em 1.5em;
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius);
    background: var(--highlight-color);

    > p:first-of-type {
      margin-top: 0;
    }
    > p:last-of-type {
      margin-bottom: 0;
    }
  }

  a:not(.button) {
    color: var(--color-iris-60);
  }

  img {
    max-width: 100%;
    border-radius: var(--border-radius);
  }

  hr {
    border: 0;
    height: 1px;
    background: var(--border-color);
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

const FeaturedDocs = styled.section`
  margin: 2em 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1em;

  @media (max-width: 600px) {
    grid-template-columns: 1fr;
  }
`;

const FeaturedDoc = styled.div`
  display: flex;
  flex-direction: column;
  align-items: start;
  padding: 1.4em 1.5em;
  background: var(--highlight-color);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);

  > h3:first-child {
    margin-top: 0;
  }

  p:last-of-type {
    flex-grow: 1; // ensure button is at the bottom
  }

  .button {
    align-self: end;
  }
`;
