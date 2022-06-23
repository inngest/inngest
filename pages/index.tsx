import { useEffect, useState } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Script from "next/script";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import IconList from "../shared/IconList";
import VideoPlayer from "../shared/VideoPlayer";
import Button from "../shared/Button";
import Callout from "../shared/Callout";
import ContentBlock from "../shared/ContentBlock";

import DiscordCTA from "../shared/Blog/DiscordCTA";

// Icons
import Github from "src/shared/Icons/Github";
import Check from "src/shared/Icons/Check";
import CLIGradient from "src/shared/Icons/CLIGradient";
import KeyboardGradient from "src/shared/Icons/KeyboardGradient";

// TODO: move these into env vars
export const INGEST_KEY =
  "BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ";

// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

export default function Home() {
  return (
    <Wrapper className="home">
      <Head>
        <title>
          Inngest → build serverless event-driven functions in minutes
        </title>
        <meta
          property="og:title"
          content="Inngest - build serverless event-driven functions in minutes"
        />
        <meta
          property="og:description"
          content="Create, deploy, and monitor event-driven serverless functions with confidence."
        />
        <meta property="og:url" content="https://www.inngest.com" />
        <meta property="og:image" content="/logo.svg" />
        <meta
          property="og:description"
          content="Build event serverless event-driven systems in seconds"
        />
        <Script src="/inngest-sdk.js" defer async></Script>
        <Script
          defer
          src="https://static.cloudflareinsights.com/beacon.min.js"
          data-cf-beacon='{"token": "e2fa9f28c34844e4a0d29351b8730579"}'
        ></Script>
      </Head>

      <Nav />

      <Hero>
        <h1>Kill Your Queues.</h1>
        <p className="hero-subheading">
          Inngest makes it simple for you to write delayed or background jobs by
          triggering functions from events
        </p>
        <p className="hero-subheading">
          <em>No infra, no config — just ship.</em>
        </p>

        <img
          className="hero-graphic"
          src="/assets/homepage/hero-graphic-june-2022.png"
          alt="How Inngest works diagram"
        />

        <IconList
          direction="vertical"
          items={[
            "Simple publishing with HTTP + JSON",
            "No SDKs needed",
            "Developer tooling for the entire workflow",
            "No boilerplate polling code",
            "Any programming language",
            "Step function support with DAGs",
          ].map((text) => ({
            icon: Check,
            text,
          }))}
        />

        <div className="hero-ctas">
          <Button
            size="medium"
            kind="primary"
            href="https://github.com/inngest/inngest-cli#installing-inngest"
          >
            <Github />
            <span className="button-text-med">Get the cli</span>
          </Button>
          <Button size="medium" kind="outline" href="/docs?ref=home-hero">
            Explore docs →
          </Button>
        </div>
      </Hero>

      <Section>
        <header>
          <h2>
            The Complete Platform For <br />
            Everything Async
          </h2>
          <p className="subheading">
            Our serverless solution provides everything you need to effortlessly
            <br />
            build and manage every type of asynchronous and event-driven job.
          </p>
        </header>

        <ContentBlock
          layout="reverse"
          heading="No queue required"
          text={
            <>
              Inngest is serverless, requiring absolutely no infrastructure to
              manage. Use our built-in scalable queuing system.{" "}
              {/* TODO - Link to something */}
            </>
          }
          // image="/assets/homepage/cli-3-commands.png"
        />
        <ContentBlock
          layout="reverse"
          heading="A real-time admin UI keeps everyone in the loop"
          text={
            <>
              The Inngest Admin UI brings full transparency to all your
              asynchronous jobs, so you can stay on top of performance,
              throughput, and more, without needing to dig through logs.
            </>
          }
          // image="/assets/homepage/cli-3-commands.png"
        />
        <ContentBlock
          layout="reverse"
          heading="Event-driven, as easy as just sending events!"
          text={
            <>
              We built all the hard stuff so you don’t have to: idempotency,
              throttling, backoff, retries, replays, job versioning, and so much
              more. With Inngest, you just write the job and we take care of the
              rest.
            </>
          }
          // image="/assets/homepage/cli-3-commands.png"
        />
      </Section>

      <Section theme="dark">
        <header>
          <h2>
            Build for <u>Builders</u>
          </h2>
          <p className="subheading">Write business logic, not boilerplate.</p>
        </header>

        <ContentBlock
          heading="Fits your workflow"
          text={
            <>
              Inngest works just like you'd hope — write your jobs alongside
              your project code, use our CLI to create new functions, mock
              queues, test and deploy your work manually or automate it with
              your favorite tool.
            </>
          }
          image="/assets/homepage/cli-3-commands.png"
          imageSize="full"
          icon={<CLIGradient />}
        />

        <ContentBlock
          layout="reverse"
          heading="Fully flexible"
          text={
            <>
              Write your jobs in any language or framework, and POST your events
              in standard JSON. If it runs in Docker, it works with Inngest,
              with zero vendor-specific libraries or boilerplate code needed.
            </>
          }
          image="/assets/homepage/language-logos.png"
          icon={<KeyboardGradient />}
        />

        <ContentBlock
          heading="Build in Minutes, Not Days"
          text={
            <>
              Zero config from setup to production — with Inngest there's no
              need to configure or manage queues, event stream topics, workers,
              or builds. Write jobs, send events, with zero fuss.
            </>
          }
          image="/assets/homepage/payload-and-job-generic.png"
          icon={<KeyboardGradient />}
        />
      </Section>

      <div className="discord-cta-wrapper">
        <DiscordCTA size="small" />
      </div>

      <Footer />
    </Wrapper>
  );
}

// Wrapper defines a top-level scope for nesting home-specific CSS classes within.
const Wrapper = styled.div`
  .section-header-top {
    margin-top: 6rem;
  }

  .button-group {
    display: flex;
    justify-content: center;
  }

  .use-cases-header {
    margin-top: 6rem;
  }
  .discord-cta-wrapper {
    margin: 4em auto;
    max-width: 600px;
  }

  .video-player {
    max-width: 1000px;
    margin: 0 auto;
    border: 1px solid var(--gray);
  }
  @media (max-width: 1040px) {
    .video-player {
      margin: 0 1em;
    }
  }
`;

const Hero = styled.header`
  padding: 14vh 0 4rem;
  text-align: center;

  h1 {
    font-size: 4.4rem;
    margin-bottom: 1.7rem;
  }

  .hero-subheading {
    margin: 1em auto;
    max-width: 540px;
    font-size: 1rem;
  }

  .hero-graphic {
    margin: 2.5rem auto;
    max-width: 748px;
  }

  .icon-list {
    margin: 2.5rem auto;
    max-width: 400px;
    max-width: fit-content;
    text-align: left; // In case text wraps
  }

  .hero-ctas {
    margin-top: 2em;
    display: flex;
    justify-content: center;
  }

  .button {
    display: inline-flex;
    font-family: var(--font-mono);
    letter-spacing: -0.5px;
  }
  .button svg {
    margin-right: 0.4rem;
  }
  .button-text-light {
    font-weight: 200;
  }
  .button-text-med {
    font-weight: 600;
  }

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
    padding: 8vh 1rem;

    > div:first-of-type {
      grid-column: 1;
    }

    .hero-graphic {
      width: 90%;
    }

    .icon-list {
      max-width: fit-content;
      padding: 0 1rem;
    }

    .hero-subheading:last-child {
      padding: 0 0 2rem;
    }

    .button {
      margin: 0.5rem !important;
    }
  }
  @media (max-width: 600px) {
    h1 {
      font-size: 2rem;
    }
    .hero-subheading {
      font-size: 0.9rem;
    }
  }
`;

const Section = styled.section<{ theme?: "dark" | "light" }>`
  margin: 0 auto;
  padding: 5rem 0;
  background-color: ${({ theme }) =>
    theme === "dark" ? "var(--black)" : "inherit"};
  color: ${({ theme }) =>
    theme === "dark" ? "var(--color-white)" : "inherit"};

  header {
    text-align: center;
  }

  h2 {
    font-size: 2.6rem;
  }

  .subheading {
    margin: 1em auto;
    max-width: 900px;
    font-size: 1rem;
    line-height: 1.6em;
  }

  @media (max-width: 800px) {
    padding: 4rem 0;
    header {
      padding: 0 2rem;
    }
  }
`;

// const ContentBlock = styled.div`
//   display: flex;
//   justify-content: center;
//   margin: 10rem 0;

//   .content {
//     margin-right: 4.5rem;
//     max-width: 490px;
//   }

//   h3 {
//     margin: 1rem 0;
//     font-size: 1.6rem;
//   }
//   p {
//     font-size: 0.9rem;
//   }
// `;

const InfoBlock = styled.div`
  grid-column: span 1;

  p {
    margin: 0.8em 0;
    font-size: 0.8em;
    color: var(--font-color-secondary);
  }
`;

const Box = styled(InfoBlock)`
  padding: 1em;
  background-color: var(--bg-color);
  border-radius: var(--border-radius);
`;

const IconBox = styled.div`
  height: 1.6em;
  width: 1.6em;
  margin-bottom: 0.5em;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--border-radius);
  background: var(--primary-color);

  svg {
    max-width: 0.7em;
    max-height: 0.7em;
  }
`;

const UseCases = styled.div`
  --spacing: 1em;

  display: grid;
  margin: 2em auto;
  max-width: var(--max-page-width);
  grid-template-columns: repeat(4, 1fr);
  grid-gap: calc(2 * var(--spacing)) var(--spacing);

  @media (max-width: 1240px) {
    margin-left: calc(2 * var(--spacing));
    margin-right: calc(2 * var(--spacing));
  }
  @media (max-width: 900px) {
    grid-template-columns: repeat(3, 1fr);
  }
  @media (max-width: 700px) {
    grid-template-columns: repeat(2, 1fr);
  }
  @media (max-width: 540px) {
    grid-template-columns: repeat(1, 1fr);
  }
`;
