import React, { useState, useEffect } from "react";
import styled from "@emotion/styled";
import { Zap, Code, Smile } from "react-feather";

import Button from "src/shared/Button";
import Nav from "src/shared/nav";
import Footer from "src/shared/footer";
import DemoBlock from "src/shared/DemoBlock";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Automate User Journeys in Minutes",
        description:
          "Build out user-behavior driven flows for your product that are triggered by events sent from your app or third party integrations.",
      },
    },
  };
}

const examples = [
  {
    title: "Handle failed payments", // Alt: Handle involuntary churn
    steps: [
      {
        icon: "/icons/brands/stripe.jpg",
        description: "Stripe Webhook Trigger",
        action: (
          <>
            When <code>charge.failed</code> is received
          </>
        ),
      },
      {
        icon: "/icons/brands/mongodb.svg",
        description: "Run custom code",
        action: "Downgrade the user's plan in the database",
      },
      {
        icon: "/icons/brands/intercom.png",
        description: "Run custom code",
        action: "Notify Customer Success team in Intercom",
      },
    ],
  },
  {
    title: "Intelligent activation drip campaign",
    steps: [
      {
        icon: "/icons/webhook.svg",
        description: "Custom Event",
        action: "When a user signs up",
      },
      {
        icon: "/icons/delay.png",
        description: "Delay",
        action: "Wait 24 hours",
      },
      {
        icon: "/icons/conditional.webp",
        description: "Conditional logic",
        action: "If user does not activate",
      },
      {
        icon: "/icons/brands/sendgrid.png",
        description: "Run custom code",
        action: "Send onboarding email",
      },
    ],
  },
  {
    title: "Running scripts from internal tools",
    steps: [
      {
        icon: "/icons/brands/retool.jpg",
        description: "Retool Resource Request",
        action: "When a form is submitted",
      },
      {
        icon: "/icons/brands/javascript.png",
        description: "Run custom code",
        action: "Run a backfill of user data",
      },
    ],
  },
];

const calloutFeatures = [
  {
    topic: "Any Source",
    title: "Connect Anything",
    description:
      "Send data via Inngest's webhooks, from your code with our API, or use one of Inngest's built-in integrations, ",
    image: "/assets/screenshots/sources.png",
    // TODO - Link to sources page (integrations, webhooks, api keys/SDKs)
  },
  {
    topic: "Developer UX",
    title: "Intuitive Developer Tooling",
    description:
      "A CLI that gets out your way and makes the hard stuff easy. Create, test, and deploy functions in minutes.",
    image: "/assets/homepage/cli-3-commands.png",
    // TODO - Link to CLI or "for developers"/developer-ux page
  },
  {
    topic: "Out-of-the-box Power",
    title: "Conditional Logic, Delays, & Automate Retries",
    description:
      "Use minimal declarative configuration to create complex flows that can delay for days, conditionally run based on data, and automatically retry failed functions.",
    image: "/assets/use-cases/conditional-logic.png",
    // TODO - Link to features page section
  },
  {
    topic: "Step Functions",
    title: "Chain Functions Together",
    description:
      "Break your code into logical steps and run them in parallel, in sequence, or conditionally based on the output of previous steps.",
    image: "/assets/use-cases/step-function.png",
    // TODO - Link to features page section on step functions
  },
];

export default function Template() {
  return (
    <>
      <Nav sticky={true} />
      <div className="hero-gradient">
        <div className="container mx-auto py-32 flex flex-row">
          <div className="text-center px-6 max-w-4xl mx-auto">
            <h1 style={{ position: "relative", zIndex: 1 }}>
              Automate{" "}
              <span className="gradient-text-ltr gradient-from-iris-60 gradient-to-cyan">
                User Journeys
              </span>
              <br />
              in Minutes.
            </h1>
            <p className="pt-6 max-w-lg mx-auto">
              Build out user-behavior driven flows for your product that are
              triggered by events sent from your app or third party
              integrations.
            </p>
            <div className="flex flex-row justify-center pt-6">
              <Button
                kind="primary"
                size="medium"
                href="/sign-up?ref=user-journey"
              >
                Sign up
              </Button>
              <Button
                kind="outline"
                size="medium"
                href="/contact?ref=user-journey"
              >
                Get a demo
              </Button>
            </div>
          </div>
        </div>
      </div>
      <div
        style={{ backgroundColor: "#f8f7fa" }}
        className="background-grid-texture"
      >
        <div className="container mx-auto max-w-5xl px-6 py-6">
          <div className="text-center px-6 max-xl mx-auto py-16">
            <h2 className="text-4xl font-normal	">
              How Customers Use Us In{" "}
              <span className="underline italic text-green-700 decoration-wavy decoration-sky-500 underline-offset-6">
                The Real World
              </span>
            </h2>
          </div>
          {examples.map((e, i) => (
            <div key={`ex-${i}`}>
              <h3 key={`title-${i}`} className="pt-6 font-normal text-xl">
                {e.title}
              </h3>
              <StepGrid
                key={`steps-${i}`}
                cols={e.steps.length}
                className="py-6"
              >
                {e.steps.map((s, j) => (
                  <Step key={`step-${j}`}>
                    <div className="icon">
                      <img src={s.icon || "x"} alt={s.description} />
                    </div>
                    <div className="text">
                      <span className="description">{s.description}</span>
                      <span className="action">{s.action}</span>
                    </div>
                  </Step>
                ))}
              </StepGrid>
            </div>
          ))}
        </div>
      </div>
      <div
        style={{ backgroundColor: "#f8f7fa" }}
        className="background-grid-texture"
      >
        <div className="container mx-auto max-w-5xl px-6 py-6">
          <div className="text-center px-6 max-xl mx-auto py-16">
            <h2 className="text-4xl font-normal	">How Inngest Works</h2>
          </div>
          <div className="grid grid-cols-3 gap-16 items-start py-6">
            <div>
              <h3 className="text-lg mb-2">Select your event trigger</h3>
              <p className="my-0">
                Configure a webhook, connect an integration, or send data
                directly from your application code with an API Key.
              </p>
            </div>
            <div>
              <h3 className="text-lg mb-2">Write business logic</h3>
              <p className="my-0">
                Use our developer tooling to quickly create, write and test your
                code to handle the event trigger.
              </p>
            </div>
            <div>
              <h3 className="text-lg mb-2">Deploy & Relax</h3>
              <p className="my-0">
                Deploy your function in seconds. Inngest runs your code every
                time a matching event trigger is received. <br />
                <em>No infra to manage.</em>
              </p>
            </div>
          </div>
          <StepGrid cols={3} className="py-6">
            <Step key="left">
              <div className="icon">
                <Zap />
              </div>
              <div className="text">
                <span className="description">Event trigger</span>
                <span className="action">Webhooks, Integrations, API Keys</span>
              </div>
            </Step>
            <Step key="mid">
              <div className="icon">
                <Code />
              </div>
              <div className="text">
                <span className="description">Business Logic</span>
                <span className="action">Any programming language</span>
              </div>
            </Step>
            <Step key="right">
              <div className="icon">
                <Smile />
              </div>
              <div className="text">
                <span className="description">Inngest Cloud</span>
                <span className="action">
                  Full history & metrics. 100% managed.
                </span>
              </div>
            </Step>
          </StepGrid>
        </div>
        <div className="text-center px-6 max-xl mx-auto py-16">
          <p className="text-2xl font-normal italic">
            <a href="/docs/quick-start?ref=user-journey-how-it-works">
              Check out the quick start guide →
            </a>
          </p>
        </div>
      </div>
      <div className="container mx-auto max-w-5xl py-24">
        <div className="text-center px-6 max-xl mx-auto pb-16">
          <h2 className="text-4xl">
            Build powerful functionality
            <br />
            <span className="gradient-text gradient-text-ltr gradient-from-cyan gradient-to-pink">
              without the overhead
            </span>
          </h2>
        </div>
        {calloutFeatures.map((f, i) => (
          <div
            key={`feature-${i}`}
            className="w-full flex flex-col lg:flex-row items-center py-4 px-8 lg:px-0"
          >
            <div
              className={`lg:w-1/2 px-6 lg:px-16 pt-8 pb-16 lg:py-0 order-2 lg:order-${
                i % 2 === 0 ? "1" : "2"
              }`}
            >
              <div className="uppercase text-color-iris-100 text-xs pb-2">
                <pre>{f.topic}</pre>
              </div>
              <h3 className="pb-2">{f.title}</h3>
              <p>{f.description}</p>
            </div>
            <div
              className={`alt-bg-${i} rounded-lg p-12 h-[350px] sm:h-[500px] lg:h-[400px] xl:h-[500px] w-full lg:w-1/2 bg-orange-50 order-1 lg:order-${
                i % 2 === 0 ? "2" : "1"
              } flex items-center justify-center overflow-hidden`}
            >
              <img src={f.image} alt={`A graphic of ${f.title} feature`} />
            </div>
          </div>
        ))}
        <div className="text-center px-6 max-xl mx-auto pt-16 flex flex justify-center">
          <Button
            kind="outlinePrimary"
            href="/sign-up?ref=user-journey-features"
          >
            Get started building now →
          </Button>
        </div>
      </div>

      <DemoBlock
        headline="Inngest provides the tools for any automation"
        description="Skip the boilerplate and get right to the heart of the matter: writing code that helps your business achieve its goals."
      />

      <Footer />
    </>
  );
}

const StepGrid = styled.div<{ cols: number | string }>`
  position: relative;
  display: grid;
  grid-template-columns: repeat(${({ cols }) => cols}, 1fr);
  grid-gap: ${({ cols }) => (cols > 3 ? "2rem" : "4rem")};

  &::before {
    position: absolute;
    z-index: 1;
    border-top: 2px dotted #b1a7b7;
    content: "";
    width: 100%;
    top: 50%;
  }
`;

const Step = styled.div`
  --spacing: 8px;

  z-index: 10;
  display: flex;
  align-items: center;
  padding: 0.8rem 1rem;
  font-size: 14px;
  background-color: var(--bg-color);
  border-radius: var(--border-radius);
  border: 2px solid var(--stroke-color);

  .icon {
    display: flex;
    justify-content: center;
    align-items: center;
    flex-shrink: 0;
    margin-right: 0.8rem;
    width: 40px;
    height: 40px;
    border-radius: var(--border-radius);
    overflow: hidden;
    text-align: center;

    img {
      max-width: 100%;
      max-height: 100%;
    }
  }

  .text {
    display: flex;
    justify-content: flex-start;
    flex-direction: column;
    gap: var(--spacing);
  }

  .description {
    font-size: 12px;
    text-transform: uppercase;
    color: var(--font-color-secondary);
  }
`;

// WIP
const TextSlider = ({ strings = [] }) => {
  const [index, setIndex] = useState(0);

  // useEffect(() => {
  //   const interval = setInterval(() => {
  //     setIndex((i) => (i + 1) % strings.length);
  //   }, 3000);
  //   return () => clearInterval(interval);
  // }, []);

  const classes = [
    "gradient-from-iris-60 gradient-to-cyan",
    "gradient-from-cyan gradient-to-pink",
    "gradient-from-pink gradient-to-orange",
  ];

  return (
    <TextSliderContainer>
      {/* <TextSliderMask /> */}
      <span style={{ position: "relative" }}>
        <TextSliderPlaceholder key="placeholder">
          {strings[0]}
        </TextSliderPlaceholder>
        <TextSliderElements
          style={{ transform: `translateY(${index * -100}%)` }}
        >
          {strings.map((s, i) => (
            <TextSliderString
              key={`string-${i}`}
              className={`gradient-text ${classes[i]} ${
                index === i
                  ? "active"
                  : index > i
                  ? "seen"
                  : index - 1 > i
                  ? "inactive"
                  : "inactive"
              }`}
            >
              {s}
            </TextSliderString>
          ))}
        </TextSliderElements>
      </span>
    </TextSliderContainer>
  );
};

const TextSliderContainer = styled.span`
  position: relative;
`;
const TextSliderMask = styled.span`
  position: absolute;
  z-index: 2;
  top: -500%;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--bg-color);
`;
const TextSliderPlaceholder = styled.span`
  visibility: hidden;
  z-index: -10;
`;
const TextSliderElements = styled.span`
  position: absolute;
  z-index: 1;
  top: 12%;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  white-space: nowrap;
  height: 100%;
  align-items: start;
  transition: all ease-in-out 200ms;
`;
const TextSliderString = styled.span`
  transition: all ease-in-out 200ms;
  opacity: 0;

  &.active {
    opacity: 1;
  }
  &.seen {
    opacity: 0.5;
  }
  & + & {
    margin-top: 1rem;
  }
`;
