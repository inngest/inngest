import Link from "next/link";
import React, { ReactNode } from "react";
import Head from "next/head";
import Image from "next/image";
import styled from "@emotion/styled";
import { Global, css } from "@emotion/react";

import DocsNav from "../shared/Docs/DocsNav";
import Footer from "../shared/legacy/Footer";
import Button from "../shared/legacy/Button";
import { getAllDocs, Categories, Sections } from "../utils/docs";
import docsSyntaxHighlightingCSS from "../shared/legacy/syntaxHighlightingCSS";
import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";

export async function getStaticProps() {
  const { sections } = getAllDocs();
  return {
    props: {
      sections,
      meta: {
        title: `Documentation & Guides`,
        description: `Learn how to use Inngest`,
      },
      htmlClassName: "docs",
      designVersion: "2",
    },
  };
}

const code = `import { createStepFunction } from "inngest";

type UserSignup = {
  name: "user/new.signup",
  data: {
    email: string;
    name: string;
  }
}

export default createStepFunction<UserSignup>("Signup flow", "user/new.signup", ({ event, tools }) => {
  // If this step fails it will retry automatically.  It will only
  // run once if it succeeds ⚡
  const promo = tools.run("Generate promo code", async () => {
    const promoCode = await generatePromoCode();
    return promoCode.code;
  });

  // Again, if the email provider is down this will retry - but we will only
  // generate one promo code ⚡
  tools.run("Send a welcome promo", async () => {
    await sendEmail({ email: event.data, promo });
  });

  // You can sleep on any platform! 😴
  tools.sleep("1 day");

  // This runs exactly 1 day after the user signs up ⏰
  tools.run("Send drip campaign", async () => {
    await sendDripCampaign();
  });
})`;

export default function DocsHome(props) {
  return (
    <DocsLayout sections={props.sections}>
      <Head>
        <title>Inngest → Documentation & Guides</title>
      </Head>
      <DocsContent hasTOC={false} className="pb-16">
        <Hero>
          <h1>Introduction</h1>
          <p>
            Inngest is an open source platform that adds superpowers to
            serverless functions.
          </p>
          <p>
            Using our SDK, a single line of code adds retries, queues, sleeps,
            cron schedules, fan-out jobs, and reliable steps to serverless
            functions in your existing projects. It's deployable to any
            platform, without any infrastructure or configuration. And,
            everything is locally testable via our UI.
          </p>
          <p>
            Learn how to get started in our{" "}
            <Link href="/docs/quick-start">quick start tutorial</Link>, or
            continue reading for an example.
          </p>
        </Hero>

        <h2>A small but powerful example</h2>

        <p>Adding sleeps, retries, and reliable steps to a function:</p>

        <SyntaxHighlighter
          language="typescript"
          showLineNumbers={false}
          style={syntaxThemeDark}
          codeTagProps={{ className: "code-window" }}
          customStyle={{
            backgroundColor: "var(--shiki-color-background)",
            fontSize: "0.7rem",
            padding: "1rem",
          }}
        >
          {code}
        </SyntaxHighlighter>

        <p>
          In this example, you can reliably run serverless functions even if
          external APIs are down. You can also sleep or delay work without
          configuring queues. Plus, all events, jobs, and functions are strictly
          typed via TypeScript for maximum correctness. Here's how things look
          when you locally run:
        </p>

        <div className="text-center">
          <Image
            src="/assets/docs/dev-server-example.png"
            alt="Inngest Dev Server screenshot"
            width={800}
            height={(609 / 900) * 800}
            quality="100"
          />
        </div>

        <h2>Comparisons</h2>

        <p>
          Without Inngest, you would have to configure several jobs amongst
          several different queues, then handle retries yourself. There's also a
          chance that many promo codes are generated depending on the
          reliability of that API. With Inngest, you can push this function live
          and everything happens automatically.
        </p>

        {/* TODO/DOCS: FUTURE: locally run this and other examples via this repo */}

        <h2>Getting started</h2>

        <p>From here, you might want to:</p>

        <ul>
          <li>
            <Link href="/docs/quick-start">
              Get started via our quick-start tutorial
            </Link>
          </li>
          <li>
            <Link href="/docs/functions">
              Learn more about functions and the tools provided
            </Link>
          </li>
          <li>
            <Link href="/docs/deploy">
              Learn how to integrate with your platform of choice
            </Link>
          </li>
          {/* TODO/DOCS: Add a link for this when we add comparisons */}
          {/* <li><Link href="/docs/">Learn more about how we compare to other tools</Link></li> */}
        </ul>

        <h2>Resources and help</h2>

        <p>
          If you have any questions we're always around in our{" "}
          <a href="/discord">Discord community</a> or on{" "}
          <a href="https://github.com/orgs/inngest/discussions">our GitHub</a>.
        </p>
      </DocsContent>
    </DocsLayout>
  );
}

export const DocsLayout: React.FC<{
  sections: { section: Sections; categories: Categories; hide: boolean }[];
  children: ReactNode;
}> = ({ children, sections }) => {
  return (
    <>
      <Global styles={DocsGlobalStyles} />
      <DocsWrapper>
        <DocsNav sections={sections} />
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
    scroll-margin-top: 2rem;
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
    margin-top: calc(3 * var(--base-size));
  }
  h3 {
    font-size: 1.3em;
    margin: calc(2 * var(--base-size)) 0 0;
  }
  h4 {
    font-size: 1.1em;
    font-weight: bold;
    margin: calc(1.5 * var(--base-size)) 0 0;
  }
  h3 + p {
    margin-top: 0.5rem !important;
  }

  // Heading links on hover
  h2,
  h3,
  h4,
  h5,
  h6 {
    position: relative;
    a {
      visibility: hidden;
      position: absolute;
      top: 0.25em;
      left: -30px;
      width: 30px;
    }
    .icon-link {
      display: block;
      height: 20px;
      width: 20px;
      mask-image: url("/assets/docs/icon-link.svg");
      background-color: var(--link-color);
    }
    &:hover a {
      visibility: visible;
    }
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

  li + li {
    margin-top: 0.5rem;
  }
  li {
    margin-left: 1rem;
  }

  hr {
    margin: 2rem 0;
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

  .button {
    --button-shadow-color: 0, 0, 0;
    --button-color: var(--color-almost-white);
    --button-border-color: var(--color-almost-black);

    border: var(--button-border-width) solid var(--button-border-color);
    border-radius: var(--border-radius);
    padding: var(--button-padding-medium);
    background: var(--button-color);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    text-decoration: none;
    text-align: center;
    transition: all 0.3s;
    white-space: nowrap;
    font-size: 0.8rem;

    color: var(--color-almost-black);
  }
  .button:hover {
    transform: translateY(-2px);
  }
  .button--primary {
    --button-shadow-color: var(--primary-color-rgb);
    --button-color: var(--primary-color);
    --button-border-color: var(--primary-color);
    color: #fff;
  }
  // For guides links with logos and copy
  .button--guide {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
    padding: 1.2rem;
    text-align: left;
    white-space: normal;
    .logo {
      height: 1.6rem;
      border-radius: 0;
    }
  }

  .button--shadow {
    box-shadow: 0 5px 25px rgba(var(--button-shadow-color), 0.6);
    &:hover {
      box-shadow: 0 5px 45px rgba(var(--button-shadow-color), 0.8);
    }
  }
  .button-icon {
    height: 0.8em;
    border-radius: 0 !important;
    margin-right: 0.4rem;
  }
  .button-container {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    margin: 1.5rem 0%;
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

  .error-text {
    text-decoration-style: wavy;
    text-underline-offset: 2px;
    text-decoration-thickness: from-font;
    text-decoration-line: underline;
    text-decoration-color: red;
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
