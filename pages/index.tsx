import { useEffect, useState } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import VideoPlayer from "../shared/VideoPlayer";
import Button from "../shared/Button";
import Callout from "../shared/Callout";
import SectionHeader from "../shared/SectionHeader";

// Icons
import ClockIcon from "../shared/Icons/Clock";
import RewindIcon from "../shared/Icons/Replays";
import PathHorizontalIcon from "../shared/Icons/PathHorizontal";
import CheckAllIcon from "../shared/Icons/CheckAll";
import FileCheckIcon from "../shared/Icons/FileCheck";
import ArchiveIcon from "../shared/Icons/Archive";
import ArrowUpRightIcon from "../shared/Icons/ArrowUpRight";

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
        <link
          rel="stylesheet"
          href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.4.0/styles/github-dark.min.css"
        />
        <script src="/inngest-sdk.js" defer async></script>
        <script
          defer
          src="https://static.cloudflareinsights.com/beacon.min.js"
          data-cf-beacon='{"token": "e2fa9f28c34844e4a0d29351b8730579"}'
        ></script>
      </Head>

      <Nav />

      <div className="grid hero-grid">
        <Hero className="grid-center-8">
          <div>
            <h1>
              Build, test, and ship reactive functions <em>in&nbsp;minutes</em>
            </h1>
            <p className="hero-subheading">
              Inngest is an <strong>event-driven serverless platform</strong>{" "}
              that lets you <strong>focus on your product</strong> by giving you
              all the tools you need to build, test, and ship reactive
              serverless functions faster than ever before.
            </p>
            <p className="hero-subheading">No infra, no config — just ship.</p>

            <div className="hero-ctas">
              <Button kind="primary" href="/sign-up?ref=home-hero">
                <span className="button-text-light">{">"}_</span>
                &nbsp;
                <span className="button-text-med">Start building</span>
              </Button>
              <Button kind="outline" href="/docs?ref=home-hero">
                Explore docs →
              </Button>
            </div>
          </div>
          <img
            src="/assets/preview.svg?v=2022-04-15"
            alt="Inngest visualization"
          />
        </Hero>
      </div>

      <div className="grid-line-horizontal"></div>

      <SectionHeader
        className="section-header-top"
        size="large"
        title={
          <>
            <span className="light-text">We help</span> developers build
            event-driven, reactive systems — faster & easier than ever.
          </>
        }
        subtitle="Our CLI and web IDE let you scaffold, develop, test, and deploy serverless functions — without any config:"
      />

      <VideoPlayer
        className="video-player"
        src="/assets/homepage/init-run-deploy-2022-04-20.mp4"
        autoPlay={true}
        duration={53}
        chapters={[
          {
            name: "Build",
            start: 0,
          },
          {
            name: "Test",
            start: 20,
          },
          {
            name: "Deploy",
            start: 29.1,
          },
        ]}
      />

      <SectionHeader
        title="Developer tooling, built specifically with good UX."
        subtitle="Everything we craft is to help you ship better and faster."
      />

      <div className="grid">
        <div className="grid-center-6 sm-col-8-center">
          <div className="button-group">
            <Button
              kind="primary"
              size="small"
              href="/sign-up?ref=home-vid-cta"
            >
              <ArrowUpRightIcon /> Start for free
            </Button>
            <Button kind="outline" size="small" href="/product">
              <ArrowUpRightIcon /> How it works
            </Button>
          </div>
        </div>
      </div>

      <Consulting>
        <p>
          <strong>Get started with experts.</strong> We're working with startups
          and engineering teams to consult implement product functionality, no
          strings attached.
        </p>
        <p className="secondary-text">
          Let us show you how to build reliable serverless functionality for
          your product, in minutes. We'll walk through implementation using your
          product requirements from start to end &mdash; delivering live
          functionality for your product.
        </p>
        <p className="secondary-text">Get in touch:</p>
        <div className="grid">
          <div className="grid-center-6 sm-col-8-center">
            <div className="button-group">
              <Button
                kind="primary"
                size="small"
                href="https://calendly.com/inngest-thb/30min"
              >
                <ArrowUpRightIcon /> Schedule a call
              </Button>
            </div>
          </div>
        </div>
      </Consulting>

      <SectionContext>
        <h3>
          Fast, reliable event-driven systems for all. With powerful
          functionality out of the box.
        </h3>
        <p className="secondary-text">
          Discover the easiest way to build scalable, complex software — by
          letting us do the infrastructure and platform for you. Here's some of
          the features you get for free:
        </p>
      </SectionContext>

      {/* Features */}
      <BoxGrid>
        <Box>
          <IconBox>
            <RewindIcon />
          </IconBox>
          <h4>Historical testing</h4>
          <p>Go one step further than integration tests.</p>
          <p>
            <strong>
              Test your serverless functions with real, historical data before
              deploying.
            </strong>{" "}
            Guarantee that your code works in the real world before it's live.
          </p>
        </Box>
        <Box>
          <IconBox>
            <ClockIcon />
          </IconBox>
          <h4>Time travel</h4>
          <p>Never wish you'd built something earlier.</p>
          <p>
            Guarantee that your code works in the real world before it's live.
            Deploy functionality then{" "}
            <strong>
              process historic events as if your feature were live in the past
            </strong>
            . Never before have you been able to make your team and users this
            happy.
          </p>
        </Box>
        <Box>
          <IconBox>
            <PathHorizontalIcon />
          </IconBox>
          <h4>Coordinated functionality</h4>
          <p>
            Wave goodbye to messy cron jobs to check whether logic should run.
          </p>
          <p>
            <strong>
              Chain multiple functions together, only running steps when
              specific events happen
            </strong>
            . Or... don't happen. No spaghetti code required.
          </p>
        </Box>
        <Box>
          <IconBox>
            <CheckAllIcon />
          </IconBox>
          <h4>Idempotency</h4>
          <p>No nightmares about building it yourself.</p>
          <p>
            <strong>
              When you need it, ensure that items are processed once
            </strong>{" "}
            — and only once. Built in, configurable idempotency for each
            function allows you to rest easy.
          </p>
        </Box>
        <Box>
          <IconBox>
            <FileCheckIcon />
          </IconBox>
          <h4>Data Enrichment</h4>
          <p>Messy data? Don't know what you mean :)</p>
          <p>
            <strong>Enrich any event with additional data</strong>, ensuring
            that your functions, data pipelines, and team have everything they
            need from the start.
          </p>
        </Box>
        <Box>
          <IconBox>
            <ArchiveIcon />
          </IconBox>
          <h4>Versioning, audits, rollbacks...</h4>
          <p>An easy way to answer “why did this happen four weeks ago?”.</p>
          <p>
            <strong>See every version of every function</strong>, the exact
            times each function was live, and which version was used for each
            event. With immediate rollbacks, when you need it.
          </p>
        </Box>
      </BoxGrid>

      <Callout
        small="Ready to get started?"
        ctaRef="home-callout-mid"
        style={{ margin: "6rem 0" }}
      />

      <SectionHeader
        title="Use cases"
        subtitle="A few examples on how you can leverage Inngest’s event-driven platform."
        align="left"
      />

      <UseCases>
        <InfoBlock>
          <h4>Webhooks</h4>
          <p>
            Process incoming webhooks as events, getting HA, event typing, and
            retries for free — no infra needed.
          </p>
        </InfoBlock>
        <InfoBlock>
          <h4>Background jobs</h4>
          <p>
            Run functions as background jobs with a single HTTP request,
            speeding up your API.
          </p>
        </InfoBlock>
        <InfoBlock>
          <h4>Coordinated logic</h4>
          <p>
            Handle logic based off of a sequence of events without crons,
            background jobs, and complex state.
          </p>
        </InfoBlock>
        <InfoBlock>
          <h4>User flows</h4>
          <p>
            Implement functionality triggered by user activity automatically, in
            any language
          </p>
        </InfoBlock>
        <InfoBlock>
          <h4>Scheduled jobs</h4>
          <p>
            Run jobs automatically, on a schedule, with full logs and
            versioning. And, of course, no infra needed.
          </p>
        </InfoBlock>
        <InfoBlock>
          <h4>Internal tools</h4>
          <p>
            Empower your team to do more with functions built for your team to
            run, with full audit trails baked in.
          </p>
        </InfoBlock>
        <InfoBlock>
          <h4>Integrations</h4>
          <p>
            Work with integrations automatically triggered by events in a single
            place — no complex app code necessary.
          </p>
        </InfoBlock>
      </UseCases>

      {/* NOTE - We'll bring this back when we add the architecture section and we have more space between callouts */}
      {/* <Callout small="Still reading?" ctaRef="home-callout-end" /> */}

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

  .hero-grid {
    padding-top: var(--nav-height);
    margin-top: calc(var(--nav-height) * -1);
    background: url(/assets/hero-grid.svg) no-repeat right 10%;
    align-items: center;
    p {
      color: #fff;
    }
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

const Consulting = styled.div`
  --stripe-color: #15151c;

  background: linear-gradient(
    135deg,
    var(--stripe-color) 0%,
    var(--bg-color) 12.5%,
    var(--bg-color) 50%,
    var(--stripe-color) 15%,
    var(--stripe-color) 15%,
    var(--bg-color) 62.5%,
    var(--bg-color) 100%
  );
  background-size: 9px 9px;

  padding: 8vh 0;
  margin: 10vh 0;
  border-top: 1px dashed var(--grid-line-color);
  border-bottom: 1px dashed var(--grid-line-color);
  text-align: center;

  p:first-of-type {
    margin: 0.2rem auto 1rem;
    font-size: 1.1em;
  }
  p {
    max-width: 800px;
    margin: 0 auto 2rem;
    text-align: center;
  }
`;

const Hero = styled.div`
  padding: 3em 0 4em;

  display: grid;
  grid-template-columns: repeat(8, 1fr);
  grid-gap: var(--grid-gap);

  > div:first-of-type {
    grid-column: span 5;
  }

  img {
    grid-column: span 3;
    width: 100%;
  }

  .hero-subheading {
    margin: 0.5em 0;
    max-width: 600px;
    font-size: 1.1rem;
  }

  .hero-ctas {
    margin-top: 2em;
  }

  .button {
    font-family: var(--font-mono);
    display: inline-block;
    letter-spacing: -0.5px;
  }
  .button-text-light {
    font-weight: 200;
  }
  .button-text-med {
    font-weight: 600;
  }

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
    padding: 8vh 0;

    > div:first-of-type {
      grid-column: 1;
    }

    img {
      display: none;
    }

    .hero-subheading:last-child {
      padding: 0 0 2rem;
    }

    .button {
      display: flex;
      align-self: stretch;
      margin: 0.5rem 0 0 0;
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

const SectionContext = styled.div`
  --stripe-color: #15151c;

  /*
  background: linear-gradient(
    135deg,
    var(--stripe-color) 12.5%,
    var(--bg-color) 12.5%,
    var(--bg-color) 50%,
    var(--stripe-color) 50%,
    var(--stripe-color) 62.5%,
    var(--bg-color) 62.5%,
    var(--bg-color) 100%
  );
  background-size: 9px 9px;
*/

  margin: 4em auto 3em;
  padding: 1.5em 3em;
  max-width: 30rem;
  text-align: center;

  h3 {
    font-size: 1.1em;
  }
  p {
    font-size: 0.8em;
  }
`;

const BoxGrid = styled.div`
  --spacing: 1em;

  display: grid;
  margin: 2em auto;
  max-width: var(--max-page-width);
  grid-template-columns: repeat(3, 1fr);
  grid-gap: var(--spacing);

  background: radial-gradient(
    51.4% 51.4% at 50% 50%,
    var(--bg-primary-highlight) 11.81%,
    var(--bg-color) 71.75%
  );

  @media (max-width: 1100px) {
    margin-left: var(--spacing);
    margin-right: var(--spacing);
    grid-template-columns: repeat(2, 1fr);
  }

  @media (max-width: 700px) {
    margin-left: var(--spacing);
    margin-right: var(--spacing);
    grid-template-columns: repeat(1, 1fr);
  }
`;

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
