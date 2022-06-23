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
import TrendingUp from "src/shared/Icons/TrendingUp";

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

        <div className="cta-container">
          <Button href="/product?ref=home-start-building" kind="primary">
            Start building today
          </Button>
        </div>
      </Section>

      <BlackBackgroundWrapper>
        <NextLevelSection>
          <header>
            <h2>
              <TrendingUp /> <span className="gradient-text">Next-Level</span>{" "}
              Async Awesomeness
            </h2>
            <p className="subheading">
              Building the future with event-driven experiences
            </p>
          </header>

          <div className="content-grid">
            <div>
              <h3>Limitless</h3>
              <p>
                Inngest jobs aren't bound by artificial time or isolation
                constraints. Develop long running, context aware tasks that
                coordinate and interact to build even the most sophisticated
                workflows.
              </p>
            </div>
            <div>
              <h3>Controlled</h3>
              <p>
                Our platform enforces data governance and accuracy so you'll
                immediately know if issues arise. Our detailed audit logs mean
                you're never in the dark.
              </p>
            </div>
            <div>
              <h3>Experienced</h3>
              <p>
                Our founding team has built high-throughput complex event-driven
                systems that scale to millions of daily events and we're excited
                to share with you the reliable performant system we always
                wished we had.
              </p>
            </div>
          </div>

          <div className="cta-container">
            <Button
              href="/product?ref=home-start-building"
              kind="outlinePrimary"
            >
              Take it to the next level <TrendingUp size="1em" />
            </Button>
          </div>
        </NextLevelSection>
      </BlackBackgroundWrapper>

      <SocialProof>
        <blockquote>
          “This is 100% the dev/prod parity that we’re lacking for queue-based
          systems.”
        </blockquote>
        <div className="attribution">
          <img src="/assets/team/dan-f-2022-02-18.jpg" />
          Developer A. - Staff Engineer at XYZ
        </div>
      </SocialProof>

      <ClosingSection>
        <header>
          <h2>Write Code, Not Too Much, Mostly Business Logic</h2>
          <p className="subheading">
            Inngest tasks lets you skip the boilerplate and get right to the
            heart of the matter:
            <br />
            writing code that helps your business achieve its goals.
          </p>
        </header>
        <div className="cta-container">
          <Button
            href="/product?ref=home-start-building"
            kind="primary"
            size="medium"
          >
            See Inngest in Action
          </Button>
        </div>
      </ClosingSection>

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

    svg {
      display: inline-block;
      margin-right: 0.1rem;
      vertical-align: top;
      position: relative;
      top: 0.2rem;
    }
  }

  .subheading {
    margin: 1em auto;
    max-width: 900px;
    font-size: 1rem;
    line-height: 1.6em;
  }

  .cta-container {
    text-align: center;

    .button {
      display: inline-flex;
    }
  }

  @media (max-width: 800px) {
    padding: 4rem 0;
    header {
      padding: 0 2rem;
    }
    h2 {
      font-size: 2rem;

      svg {
        display: block;
        margin: 0 auto;
      }
    }
  }
`;

const BlackBackgroundWrapper = styled.div`
  background: linear-gradient(180deg, black 50%, transparent 50%);
`;

const NextLevelSection = styled(Section)`
  width: 96%;
  max-width: 1200px;
  padding: 2.5rem;

  background: linear-gradient(134.83deg, #f4f4fb 24.75%, #fbfbff 89.21%);
  box-shadow: 0px 2px 20px rgba(0, 0, 0, 0.25);
  border-radius: 20px;

  .gradient-text {
    background: linear-gradient(180deg, #5d5fef 0%, #ef5da8 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }

  .content-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    grid-gap: 2rem;
    margin: 5rem 0;

    h3 {
      margin-bottom: 1rem;
      font-style: italic;
    }
  }

  @media (max-width: 960px) {
    .content-grid {
      margin: 3rem 0;
      grid-template-columns: repeat(5, 1fr);

      > div:nth-child(1) {
        grid-column: 1/4;
      }
      > div:nth-child(2) {
        grid-column: 2/5;
      }
      > div:nth-child(3) {
        grid-column: 3/6;
      }
    }
  }
  @media (max-width: 800px) {
    padding: 2rem 1rem;
    .content-grid {
      display: flex;
      padding: 0 1rem;
      flex-direction: column;
    }
  }
`;

const ClosingSection = styled(Section)`
  h2 {
    font-size: 2.1rem;
  }
  .cta-container {
    margin-top: 3rem;
  }
`;

const SocialProof = styled.section`
  max-width: 800px;
  margin: 20vh auto 10vh;
  padding: 0 1rem;
  text-align: center;

  blockquote {
    font-size: 1.6rem;
    font-style: italic;
    font-weight: bold;
    color: var(--color-gray-purple);
  }
  .attribution {
    display: inline-flex;
    align-items: center;
    margin-top: 1rem;
    font-size: 0.8rem;
  }
  img {
    height: 1.4rem;
    width: 1.4rem;
    border-radius: 1rem;
    margin-right: 0.6rem;
  }

  @media (max-width: 800px) {
    margin: 14vh auto 8vh;
  }
`;
