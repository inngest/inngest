import { useEffect, useState } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Button from "../shared/Button";
import Code from "../shared/Code";
import Callout from "../shared/Callout";
import Integration, { IntegrationType } from "../shared/Integration";
import SectionHeader from "../shared/SectionHeader";

// Icons
import ClockIcon from "../shared/Icons/Clock";
import RewindIcon from "../shared/Icons/Replays";
import PathHorizontalIcon from "../shared/Icons/PathHorizontal";

import Hub from "../shared/Icons/Hub";
import Functions from "../shared/Icons/Functions";

import Create from "../shared/Icons/Create";
import Deploy from "../shared/Icons/Deploy";
import Monitor from "../shared/Icons/Monitor";
import CLI from "../shared/Icons/CLI";
import Retries from "../shared/Icons/Retries";

import Transforms from "../shared/Icons/Transforms";
import Versions from "../shared/Icons/Versions";
import Rollback from "../shared/Icons/Rollback";
import Audit from "../shared/Icons/Audit";
import Logging from "../shared/Icons/Logging";

import Observability from "../shared/Icons/Observability";
import Alerting from "../shared/Icons/Alerting";

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
        <link rel="icon" href="/favicon.png" />
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
              Build, test, and ship reactive functions <br />
              <em>in minutes</em>
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

      <div className="grid">
        <div className="grid-center-6 sm-col-8-center">
          <img
            src="/assets/homepage/cli-ui-placeholder.svg"
            alt="Using our CLI and Web IDE"
            className="full-width"
          />
        </div>
      </div>

      <SectionHeader
        title="Developer tooling, built specifically with good UX."
        subtitle="Everything we craft is to help you ship better and faster."
      />

      <div className="grid">
        <div className="grid-center-6 sm-col-8-center">
          <div className="button-group">
            <Button kind="primary" size="small">
              Start for free
            </Button>
            <Button kind="outline" size="small">
              What is Inngest?
            </Button>
          </div>
        </div>
      </div>

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
          <p>Never wish you’d built something earlier.</p>
          <p>
            Guarantee that your code works in the real world before it's live.
            Deploy functionality then{" "}
            <strong>
              process historic events as if your feature were live in the past
            </strong>
            . Never before have you been able to make your team and users
            this happy.
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
            <PathHorizontalIcon /> {/* TODO */}
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
            <PathHorizontalIcon /> {/* TODO */}
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
            <PathHorizontalIcon /> {/* TODO */}
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

      <Callout small="Ready to get started?" ctaRef="home-callout-mid" />

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

      <Callout small="Still reading?" ctaRef="home-callout-end" />

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

  margin: 3em auto 2em;
  padding: 1.5em 3em;
  max-width: 30rem;
  text-align: center;
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
    max-width: 0.5em;
    max-height: 0.5em;
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
