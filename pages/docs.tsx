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

        <div className="grid pt-4 gap-4 sm:grid-cols-1 xl:grid-cols-3">
          <Quickstart
            href="/docs/functions"
            className="shadow-md rounded-sm p-4 border-2 hover:shadow-2xl"
          >
            <p className="text-base my-2 text-color-primary">
              👩‍💻 &nbsp; Writing functions
            </p>
            <p className="text-color-secondary">
              Learn how to write functions using Typescript or Javascript using any
              platform or framework
            </p>
          </Quickstart>
          <Quickstart
            href="/docs/events"
            className="shadow-md rounded-sm p-4 border-2 hover:shadow-2xl"
          >
            <p className="text-base my-2 text-color-primary">
              📢 &nbsp; Sending events
            </p>
            <p className="text-color-secondary">
              Learn how to trigger background jobs by sending events from your code
            </p>
          </Quickstart>
          <Quickstart
            href="/docs/deploy"
            className="shadow-md rounded-sm p-4 border-2 hover:shadow-2xl"
          >
            <p className="text-base my-2 text-color-primary">🚢 &nbsp; Deploying</p>
            <p className="text-color-secondary">
              Deploy functions to your platform of choice, such as Vercel, Netlify,
              Cloudflare, or AWS
            </p>
          </Quickstart>
        </div>

        <h2 className="pt-6">What is Inngest?</h2>

        <p>Inngest is a serverless platform that allows you to build, test, and deploy serverless background functions and scheduled tasks — without any infrastructure, queues, or stateful long-running services.</p>

        <p>
Using Inngest you can write serverless functions triggered by events within your existing code, zero boilerplate or infra required.</p>

        <h2 className="pt-4">Key features</h2>
        <ul>
        <li><strong>Fully serverless:</strong>  Run background jobs, scheduled functions, and build event-driven systems without any servers, state, or setup</li>
<li><strong>Deploy anywhere</strong>:  works with NextJS, Netlify, Vercel, Redwood, Express, Cloudflare, and Lambda</li>
<li><strong>Use your existing code:</strong>  write functions within your current project, zero learning required</li>
<li><strong>A complete platform</strong>:  complex functionality built in such as event replay, canary deploys, version management and git integration</li>
<li><strong>Fully typed</strong>:  Event schemas, versioning, and governance out of the box</li>
<li><strong>Observable</strong>:  A full UI for managing and inspecting your functions</li>
<li><strong>Any language:</strong>  Use our CLI to write functions using any language</li>
        </ul>

        <h2 className="pt-4">How it works</h2>
        <p>
        Inngest accepts events from your system, then runs any functions which listen to those events in parallel, with built retries if things fail.  Events are JSON objects sent via POST request and can be triggered from your own code, from webhooks, or from integrations.</p>

        <h2 className="pt-4">Use cases</h2>
        <p>
          Inngest users are typically developers and data engineers. They use
          Inngest to reliably run background work, serverless functions, and
          scheduled jobs across a variety of use cases. Common examples include:
        </p>

        <ul>
          <li>
            <p>
              <b>Building reliable webhooks</b>
              <br />
              Inngest acts as a layer which can handle webhook events and that run your functions automatically. The Inngest Cloud dashboard gives your complete observability into what event payloads were received and how your functions ran.
            </p>
          </li>
          <li>
            <p>
              <b>Serverless background jobs</b>
              <br />
              Ensure your API is fast by running your code, asynchronously, in the background, without queues or long-running workers. Background jobs are triggered by events and have built in retries and logging.
            </p>
          </li>
          <li>
            <p>
              <b>Scheduled jobs</b>
              <br />
              Run your function on a schedule to repeat hourly, daily, weekly or whatever you need.
            </p>
          </li>
          <li>
            <p>
              <b>Internal tools</b>
              <br />
              Trigger scripts in your code to run from your own internal tools or third party products like Retool.
            </p>
          </li>
          <li>
            <p>
              <b>User journey automation</b>
              <br />
              Use customer behavior events to trigger automations to run like drip email campaigns, re-activation campaigns, or reminders.
            </p>
          </li>
          <li>
            <p>
              <b>Event-driven systems</b>
              <br />
              Developers can send and subscribe to a variety of internal and external events, creating complex event-driven architectures without worrying about infrastructure, SDKs, and boilerplate.
            </p>
          </li>
          <li>
            <p>
              <b>
              Complex pipelines & workflows
              </b>
              <br />
Build multi-step pipelines and workflows using conditional logic, delays or multiple events.
            </p>
          </li>
        </ul>

        <h2 className="pt-4">Ready to get started?</h2>
        <p className="pb-6">Learn how to write functions in your project within a few seconds</p>

        <Button
          kind="primary"
          size="small"
          href="/sign-up?ref=docs-started"
          style={{ display: "inline-block" }}
        >
          Get started
        </Button>

        {/*
        <Button
          kind="black"
          size="small"
          href="/quick-starts?ref=docs-started"
          style={{ display: "inline-block" }}
        >
          See quick-starts
        </Button>
        */}
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
  gap: 4em 6em;
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
    margin-top: calc(4 * var(--base-size));
  }
  h3 {
    font-size: 1.3em;
    margin: calc(2.5 * var(--base-size)) 0 0;
  }
  h3 + p {
    margin-top: 0.5rem !important;
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

  ol ul {
    margin: 0.25rem 0 0 1.5rem;
  }

  ul {
    list-style-type: disc;
    margin-left: 1rem;
  }
  ol {
    list-style-type: number;
  }

  li + li { margin-top: 0.5rem }
  li {
    margin-left: 1rem;
  }

  aside,
  video {
    margin: 2rem 0;
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

const Quickstart = styled.a`
  background: var(--highlight-color);
  border-color: var(--border-color);
  color: var(--font-color-primary);
`;

const FeaturedDoc = styled.div`
  display: flex;
  flex-direction: column;
  align-items: start;
  padding: 1.4em 1.5em;

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
