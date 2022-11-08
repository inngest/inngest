import { useEffect, useState } from "react";
import styled from "@emotion/styled";
import Head from "next/head";

import Nav from "../shared/nav";
import Footer from "../shared/Footer";

import Section from "../shared/Section";
import ContentBlock from "../shared/ContentBlock";
import IconList from "../shared/IconList";
import Check from "src/shared/Icons/Check";
import Button from "src/shared/Button";
import CLIInstall from "src/shared/CLIInstall";

// TODO: move these into env vars
export const INGEST_KEY =
  "BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ";

// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Product - Features & Functionality",
        description:
          "The features and functionality that get out of your way and let you build.",
        image: "/assets/product/queue-checkmark.png",
      },
    },
  };
}

export default function Home() {
  return (
    <Wrapper className="home">
      <Nav />
      <Hero>
        <h1>
          No&nbsp;config. No&nbsp;Infra.{" "}
          <span className="gradient-text">Just&nbsp;Ship.</span>
        </h1>
        <p>
          The features and functionality that get out of your way and let you
          build.
        </p>

        <CLIInstall />
      </Hero>

      <EventsSection>
        <header>
          <h2>Start sending events in seconds</h2>
        </header>

        <div className="events-feature-list">
          {[
            {
              image: "/assets/product/queue-checkmark.png",
              heading: "No Queues to configure",
              text: "Create a new API key and your good to go.",
            },
            {
              image: "/assets/product/http-request-libs.png",
              heading: "No SDKs needed",
              text: "Send events with just HTTP and JSON. Use your standard lib or your favorite request library.",
            },
            {
              image: "/assets/product/event-schema-type.png",
              heading: "Auto-generated event schemas",
              text: "Data governance out of the box lets you understand you write predictable background jobs.",
            },
          ].map(({ heading, text, image }) => (
            <>
              <img src={image} alt={heading} />
              <div className="events-feature-item">
                <h3>{heading}</h3>
                <p>{text}</p>
              </div>
            </>
          ))}
        </div>
        <div className="cta-container">
          <Button
            href="/docs/event-format-and-structure?ref=product-send-events"
            kind="outline"
            size="medium"
          >
            Learn about events
          </Button>
        </div>
      </EventsSection>

      <Section>
        <header>
          <h2>Write code, not workers</h2>
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
          image="/assets/product/declarative-functions.png"
          imageSize="full"
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
          image="/assets/product/no-polling.png"
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
          image="/assets/product/user-flow.png"
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
          image="/assets/product/function-versions.png"
          imageSize="full"
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
                <a href="/docs/reference/functions/retries?ref=product-feature-list">
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
            Create a function in 2 minutes
          </Button>
        </div>
      </Section>

      <Section theme="dark">
        <header>
          <h2>
            <span className="gradient-text">{"\u276f"}</span> A CLI designed for
            your workflow
          </h2>
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
          image="/assets/product/cli-init.png"
          imageSize="full"
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
          image="/assets/product/cli-run.png"
          imageSize="full"
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
          image="/assets/product/cli-dev.png"
          imageSize="full"
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
          image="/assets/product/cli-deploy.png"
          imageSize="full"
        />

        <div className="cta-container">
          <CLIInstall />
        </div>
      </Section>

      <Section>
        <header>
          <h2>The Choice is Yours</h2>
          <p className="subheading">Self-host or lets us do it</p>
        </header>
        <PlatformGrid>
          <PlatformBox>
            <h3>Cloud</h3>
            <IconList
              direction="vertical"
              items={[
                "Fully managed for your team",
                "From idea to production in minutes",
              ].map((text) => ({
                icon: Check,
                text,
              }))}
            />
            <Button
              href="/sign-up?ref=product-choice"
              kind="primary"
              size="medium"
            >
              Sign up today
            </Button>
          </PlatformBox>
          <PlatformBox>
            <h3>Self-hosted</h3>
            <IconList
              direction="vertical"
              items={["Customize for your needs", "Deploy to any cloud"].map(
                (text) => ({
                  icon: Check,
                  text,
                })
              )}
            />
            <Button
              href="/docs/self-hosting?ref=product-choice"
              kind="outline"
              size="medium"
            >
              Learn how
            </Button>
          </PlatformBox>
        </PlatformGrid>
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
      padding: 0 1rem;
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

  .cli-install {
    margin: 2rem;
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

const EventsSection = styled(Section)`
  padding-right: 4rem;
  padding-left: 4rem;

  background: linear-gradient(
    135deg,
    hsl(332deg 30% 95%) 0%,
    hsl(240deg 30% 95%) 100%
  );

  .events-feature-list {
    display: grid;
    grid-template-columns: 300px 1fr;
    grid-gap: 0 1rem;
    align-items: center;
    max-width: 840px;
    margin: 2rem auto;
    padding: 0 1rem;
  }
  .events-feature-item {
    h3 {
      margin-bottom: 1rem;
    }
  }

  @media (max-width: 660px) {
    .events-feature-list {
      grid-template-columns: auto;
      grid-row-gap: 1rem;
      padding: 0 10%;
      img {
        margin: 0 auto;
        height: 180px;
      }
    }
  }
`;

const PlatformGrid = styled.div`
  display: grid;
  margin: 3rem auto;
  padding: 0 1rem;
  max-width: 840px;
  grid-template-columns: 1fr 1fr;
  grid-gap: 1rem;

  @media (max-width: 640px) {
    grid-template-columns: 1fr;
  }
`;

const PlatformBox = styled.div`
  display: flex;
  flex-direction: column;
  padding: 2rem;
  background: linear-gradient(
    135deg,
    hsl(332deg 30% 95%) 0%,
    hsl(240deg 30% 95%) 100%
  );
  border-radius: var(--border-radius);

  h3 {
    font-size: 1.5rem;
  }

  .icon-list {
    margin: 1.8rem 0;
    flex-grow: 1;
  }
`;
