import Link from "next/link";
import React from "react";
import Head from "next/head";
import styled from "@emotion/styled";
import { Global, css } from "@emotion/react";

import DocsNav from "../shared/Docs/DocsNav";
import Footer from "../shared/footer";
import Button from "../shared/Button";
import ArrowUpRightIcon from "../shared/Icons/ArrowUpRight";
import { getAllDocs, Categories } from "../utils/docs";
import docsSyntaxHighlightingCSS from "../shared/syntaxHighlightingCSS";

export async function getStaticProps() {
  const { cli, cloud } = getAllDocs();
  return { props: { cli: cli, cloud: cloud, htmlClassName: "docs" } };
}

export default function DocsHome(props) {
  return (
    <DocsLayout cli={props.cli} cloud={props.cloud}>
      <Head>
        <title>Inngest → Documentation & Guides</title>
      </Head>
      <DocsContent hasTOC={false} className="pb-16">
        <Hero>
          <h1>Inngest documentation</h1>
          <p>
            This documentation will help you become an expert in event-driven
            serverless functions within minutes.
          </p>
        </Hero>

        <h2>Quick start</h2>

        <div className="grid pt-4 grid-cols-3 gap-4">
          <a
            href="/"
            className="shadow-md rounded-sm p-4 border-slate-200 border-2 color-inherit hover:shadow-2xl bg-white"
          >
            <p className="text-base my-2">🚀 Installation</p>
            <p className="text-slate-600">
              Get up and running with the CLI in seconds. You'll have a full
              local development environment ready to go.
            </p>
          </a>
          <a
            href="/"
            className="shadow-md rounded-sm p-4 border-slate-200 border-2 color-inherit hover:shadow-2xl bg-white"
          >
            <p className="text-base my-2">👩‍💻 Writing & running fns</p>
            <p className="text-slate-600">
              Write and locally run your first function using any language and
              the CLI, triggered automatically by events
            </p>
          </a>
          <a className="shadow-md rounded-sm p-4 border-slate-200 border-2 color-inherit hover:shadow-2xl bg-white">
            <p className="text-base my-2">🚢 Deploying</p>
            <p className="text-slate-600">
              Learn how to deply your functions to production instantly, without
              managing infrastructure
            </p>
          </a>
        </div>

        <h2 className="pt-8">What is Inngest?</h2>

        <p>
          Inngest is an open-source, event-driven platform which makes it easy
          for developers to build, test, and deploy scheduled tasks and
          background functions — without worrying about infrastructure, queues,
          or stateful services.
        </p>
        <p>
          Using Inngest, you can write and deploy serverless step functions
          which are triggered by events or on a schedule without writing any
          boilerplate code or infra.
        </p>

        <h2 className="pt-8">Key features</h2>
        <ul>
          <li>CLI with developer-friendly APIs and local testing</li>
          <li>Stateless serverless step functions which run any language</li>
          <li>
            Complex functionality built in, such as event replay, canary
            deploys, version management and git integration
          </li>
          <li>Event coordination for building complex interactive functions</li>
          <li>Event governance, schemas, and forwarding out of the box</li>
        </ul>

        <h2 className="pt-8">Use cases</h2>
        <p>
          Inngest users are typically developers and data engineers. They use
          Inngest to reliably run background work, serverless functions, and
          scheduled jobs across for a variety of use cases. Common examples
          include:
        </p>

        <ul>
          <li>
            <p>
              <b>Reliably managing incoming webhooks without infrastructure</b>
              <br />
              Inngest handles any number of incoming webhooks at scale with zero
              infra, and idempotently schedules serverless functions to respond
              to webhooks without any configuration.
            </p>
          </li>
          <li>
            <p>
              <b>Serverless background jobs</b>
              <br />
              Inngest schedules background jobs from in-app events without any
              infrastructure, safely managing retries, queues, rolling deploys,
              function versioning, and event versioning automatically.
            </p>
          </li>
          <li>
            <p>
              <b>Internal tools</b>
              <br />
              Automate internal processes with versioned step functions written
              in any language, and connect functions to external platforms such
              as Retool.
            </p>
          </li>
          <li>
            <p>
              <b>Managing complex data pipelines</b>
              <br />
              Inngest can run complex data pipelines using a mixture of
              languages on a schedule or in realtime, with full local testing
              and reproducibility built in.
            </p>
          </li>
          <li>
            <p>
              <b>Building event-driven systems</b>
              <br />
              Developers can send and subscribe to a variety of internal and
              external events, creating complex event-driven architectures
              without worrying about infrastructure, SDKs, and boilerplate.
            </p>
          </li>
          <li>
            <p>
              <b>Real-time sync</b>
              <br />
              Inngest can integrate with a variety of platforms to enable
              real-time ETL and real-time reverse ETL.
            </p>
          </li>
          <li>
            <p>
              <b>Data warehousing & event federation</b>
              <br />
              Inngest can aggregate events from a variety of sources, forwarding
              them to data warehouses and other systems in addition to running
              application logic.
            </p>
          </li>
        </ul>

        <h2 className="pt-8">Ready to get started?</h2>
        <p className="pb-4">
          Learn how to install our CLI and write your first serverless function
          in minutes, then get started with our cloud and deploy your functions
          for free
        </p>

        <Button
          kind="primary"
          size="small"
          href="/sign-up?ref=docs-started"
          style={{ display: "inline-block" }}
        >
          Get started
        </Button>
      </DocsContent>
    </DocsLayout>
  );
}

export const DocsLayout: React.FC<{ cli: Categories; cloud: Categories }> = ({
  children,
  cli,
  cloud,
}) => {
  return (
    <>
      <Global styles={DocsGlobalStyles} />
      <DocsWrapper>
        <DocsNav cli={cli} cloud={cloud} />
        <Main>{children}</Main>
      </DocsWrapper>
      <Footer />
    </>
  );
};

const DocsGlobalStyles = css`
  // Push the page down on mobile below the fixed nav for the page text OR PageBanner
  .docs body {
    @media (max-width: 800px) {
      margin-top: 62px;
    }
  }
`;

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
    padding-top: 4em;
  }
  @media (max-width: 800px) {
    padding: 4em 2em 2em;
  }
`;

const Hero = styled.div`
  h1 {
    font-size: 2.5em;
    margin-bottom: 1em;
  }
`;

export const DocsContent = styled.article<{ hasTOC: boolean }>`
  --base-size: 16px;

  max-width: 900px;

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

    // Ensure text does not go under floating TOC
    @media (max-width: 1000px) {
      margin-right: ${({ hasTOC }) => (hasTOC ? "4em" : "0")};
    }
    @media (max-width: 800px) {
      margin-right: 0;
    }
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

  p:not([class*="text-base"]),
  ol,
  ul,
  li {
    font-size: 16px;
    line-height: 1.7em;
  }

  p:not([class*="my-"]) {
    margin: 1em 0;
  }

  ol,
  ul {
    margin: 1.5em 0;
  }

  ul {
    list-style-type: disc;
    margin-left: 1rem;
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

  p,
  li {
    code {
      color: var(--color-iris-100);
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

const Quickstart = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1em;
`;

const FeaturedDoc = styled.div`
  display: flex;
  flex-direction: column;
  align-items: start;
  padding: 1.4em 1.5em;
  background: var(--highlight-color);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);

  > h3:first-of-type {
    margin-top: 0;
  }

  p:last-of-type {
    flex-grow: 1; // ensure button is at the bottom
  }

  .button {
    align-self: end;
  }
`;
