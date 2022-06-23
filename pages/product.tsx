import { useEffect, useState } from "react";
import styled from "@emotion/styled";

import Nav from "../shared/nav";
import Footer from "../shared/footer";

import Section from "../shared/Section";
import ContentBlock from "../shared/ContentBlock";
import IconList from "../shared/IconList";
import Check from "src/shared/Icons/Check";
import Button from "src/shared/Button";

// TODO: move these into env vars
export const INGEST_KEY =
  "BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ";

// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

export default function Home() {
  return (
    <Wrapper className="home">
      <Nav />
      <Hero>
        <h1>
          No config. No Infra. <span className="gradient-text">Just Ship.</span>
        </h1>
        <p>
          The features and functionality that get out of your way and let you
          build.
        </p>
      </Hero>

      <Section theme="dark">
        <header>
          <h2>Start sending events in seconds</h2>
        </header>
        <div className="content-grid">
          <div>
            <h3>No Queues to configure</h3>
            <p>
              Create new API keys in a click and send events with HTTP + JSON
              (No SDKs needed).
            </p>
          </div>
          <div>
            <h3>No Queues to configure</h3>
            <p>
              Create new API keys in a click and send events with HTTP + JSON
              (No SDKs needed).
            </p>
          </div>
          <div>
            <h3>Auto-generated event schemas</h3>
            <p>
              Data governance out of the box lets you understand you write
              predictable background jobs.
            </p>
          </div>
        </div>
      </Section>

      <Section>
        <header>
          <h2>Skip the workers</h2>
        </header>
        <ContentBlock
          heading="Declarative background jobs"
          text={
            <>
              Write background jobs as functions and declare what events trigger
              them or when they will run. Your code is decoupled from the queue
              so you can deploy new functionality any time.
            </>
          }
          image="/assets/homepage/language-logos.png"
        />
        <ContentBlock
          layout="reverse"
          heading="Serverless functions"
          text={
            <>
              No need to create stateful, long-running workers that poll a queue
              - Inngest calls your functions when needed.
            </>
          }
          image="/assets/homepage/language-logos.png"
        />
        <ContentBlock
          heading="Anything that runs in a container"
          text={
            <>
              Use any programming language. Bring existing code if you want.
              Just read the payload from args and write to stdout - It's that
              easy.
            </>
          }
          image="/assets/product/dockerfile.png"
          imageSize="full"
        />
        <ContentBlock
          layout="reverse"
          heading="Simple to Sophisticated"
          text={
            <>
              Run simple background jobs or long running, multi-step,
              conditional workflows.
            </>
          }
          image="/assets/product/dockerfile.png"
        />
        <ContentBlock
          heading="Versioning built-in"
          text={
            <>
              All functions are versioned any time they are deployed making it
              easy to diagnose issues, rollback, or{" "}
              <span className="badge">COMING SOON</span> blue-green & canary
              deploys.
            </>
          }
          image="/assets/product/dockerfile.png"
        />

        <div className="everything-else-list">
          <h3>Everything else you need...</h3>

          <IconList
            direction="vertical"
            items={[
              <>
                <strong>Idempotency</strong> - Ensure functions run once - and
                only once.
              </>,
              <>
                <strong>Throttling</strong> - Limit how frequently a job can be
                run
              </>,
              <>
                <strong>Automatic Retries</strong> - Use HTTP status codes to
                define what code should be retried (
                <a href="/docs/functions/retries?ref=product-feature-list">
                  docs
                </a>
                )
              </>,
              <>
                <strong>Backoff</strong> - Exponential backoff with jitter by
                default
              </>,
            ].map((text) => ({
              icon: Check,
              text,
            }))}
          />
        </div>

        <div className="cta-container">
          <Button
            href="/docs/quick-start?ref=product-feature-list"
            kind="primary"
          >
            Learn how to create a function in 2 minutes
          </Button>
        </div>
      </Section>

      <Section theme="dark">
        <header>
          <h2>A CLI designed for your workflow</h2>
        </header>
        <ContentBlock
          preline={
            <>
              <pre className="cli-command">{"\u276f"} inngest init</pre>
            </>
          }
          heading="Create functions"
          text={
            <>
              Quickly scaffold new functions with our language templates and
              generate language types using event schemas.
            </>
          }
          image="/assets/homepage/language-logos.png"
        />
        <ContentBlock
          preline={
            <>
              <pre className="cli-command">{"\u276f"} inngest run</pre>
            </>
          }
          heading="Run with test data"
          text={
            <>
              Run your functions individually for rapid development and testing
              using test event payloads generated from event schemas.
            </>
          }
          image="/assets/homepage/language-logos.png"
        />
        <ContentBlock
          preline={
            <>
              <pre className="cli-command">{"\u276f"} inngest dev</pre>
            </>
          }
          heading="Test everything end-to-end"
          text={
            <>
              Our DevServer loads all of your functions and spins up a local
              source API so you can send events and test the entire Inngest
              stack end-to-end.
            </>
          }
          image="/assets/homepage/language-logos.png"
        />
        <ContentBlock
          preline={
            <>
              <pre className="cli-command">{"\u276f"} inngest deploy</pre>
            </>
          }
          heading="Ship your code"
          text={
            <>
              Deploys shouldn’t be an afterthought. A single command to push
              your code live to production or a test environment.
            </>
          }
          image="/assets/homepage/language-logos.png"
        />
      </Section>

      <Footer />
    </Wrapper>
  );
}

// Wrapper defines a top-level scope for nesting home-specific CSS classes within.
const Wrapper = styled.div`
  .everything-else-list {
    margin-bottom: 4rem;
    h3 {
      text-align: center;
    }
    .icon-list {
      margin: 2rem auto;
      width: fit-content;
    }
  }

  .cli-command {
    color: var(--color-gray-purple);
  }
`;

const Hero = styled.div`
  margin: 6rem 0;
  padding: 0 1rem;
  text-align: center;

  h1 {
    font-size: 2.6rem;
  }
  p {
    margin: 1rem auto;
  }

  .gradient-text {
    background: linear-gradient(
      -45deg,
      var(--green) 0%,
      var(--color-iris-100) 50%,
      var(--primary-color) 75%,
      var(--color-fuschia-100) 100%
    );
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;

    background-size: 400% 400%;
    animation: gradient 4s ease infinite;
  }

  @keyframes gradient {
    0% {
      background-position: 0% 50%;
    }
    50% {
      background-position: 100% 50%;
    }
    100% {
      background-position: 0% 50%;
    }
  }

  @media (max-width: 800px) {
  }
`;
