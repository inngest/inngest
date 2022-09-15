import React, { useEffect, useState } from "react";
import styled from "@emotion/styled";

import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Hero from "src/shared/Hero";
import Examples from "src/shared/Examples";
import HowItWorks from "src/shared/HowItWorks";
import FeatureCallouts from "src/shared/FeatureCallouts";
import DemoBlock from "src/shared/DemoBlock";
import GraphicCallout from "src/shared/GraphicCallout";

import { Hero as SDKHero } from "./features/sdk";
// import { Experiment, FadeIn } from "src/shared/Experiment";

// TODO: move these into env vars
export const INGEST_KEY =
  "BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ";

// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "You Send Events. We Run Your Code.",
        description:
          "Quickly build, test and deploy code that runs in response to events or on a schedule — without spending any time on infrastructure.",
        image: "/assets/img/og-image-default.jpg",
      },
    },
  };
}

// TODO - Use different examples for various use case, perhaps cycling through them
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

const useCases = [
  {
    icon: "/assets/homepage/icon-user-journey.png",
    href: "/uses/user-journey-automation?ref=homepage",
    title: "User Journey Automation",
    description:
      "User-behavior driven flows for your product that are triggered by events sent from your app or third party integrations.",
  },
  {
    icon: "/assets/homepage/icon-background-jobs.png",
    href: "/uses/serverless-node-background-jobs?ref=homepage",
    title: "Background Jobs",
    description:
      "Build, test, then deploy background jobs and scheduled tasks without worrying about infrastructure or queues — so you can focus on your product.",
  },
  {
    icon: "/assets/homepage/icon-internal-tools.png",
    href: "/uses/internal-tools?ref=homepage",
    title: "Internal Tools",
    description:
      "Create internal apps using any language, with full audit trails, human in the loop tasks, and automated flows.",
  },
];

export default function Home() {
  // TEMP for SDK hero
  const [language, setLanguage] = useState<"javascript" | "typescript">(
    "javascript"
  );
  const ext = language === "typescript" ? "ts" : "js";
  return (
    <div className="home">
      <Nav sticky={true} />

      <Hero
        headline={
          <>
            Build
            <br />
            <TextSlider
              strings={[
                "User Journeys",
                "Webhooks",
                "Internal Tools",
                "Background Jobs",
              ]}
            />
            <br />
            in Minutes
          </>
        }
        subheadline={
          <>
            Inngest is a developer platform for building, testing and deploying
            code that runs in response to events or on a schedule — without
            spending any time on infrastructure.
          </>
        }
        primaryCTA={{
          href: "/sign-up?ref=homepage-hero",
          text: "Get started for free →",
        }}
        secondaryCTA={{
          href: "/docs?ref=homepage-hero",
          text: "Read the docs",
        }}
      />

      <HowItWorks />

      <SDKHero
        cta={{ href: "/features/sdk?ref=homepage-sdk", text: "Learn more →" }}
        language={language}
        ext={ext}
        onToggle={(l) => setLanguage(l)}
      />

      <Examples
        heading={
          <>
            How Customers Use Us In{" "}
            <span className="underline italic text-green-700 decoration-wavy decoration-sky-500 underline-offset-6">
              The&nbsp;Real&nbsp;World
            </span>
          </>
        }
        examples={examples}
        cta={{
          href: "/quick-starts?ref=homepage-examples",
          text: "Check out our project quick starts →",
        }}
      />

      <div className="mx-auto max-w-5xl my-24">
        <div className="text-center px-6 max-w-2xl mx-auto">
          <h2 className="text-4xl mb-6">
            <span className="gradient-text gradient-text-ltr gradient-from-pink gradient-to-orange">
              Why
            </span>{" "}
            use Inngest?
          </h2>
          <p className="text-md">
            Inngest has helped engineering teams save months of dev time
            building out their products.
          </p>
          <p className="text-md">
            Inngest enables developers to quickly build out functionality
            without having to spend time or money on infrastructure and setting
            up queues, workers, retry policies, or logging. Our platform gives
            you the tooling and observability to fix issues fast, and Inngest's
            step functions enable complex workflows without having to manage
            state.
          </p>
        </div>
      </div>

      {/* Use cases */}
      <div className="mx-auto max-w-5xl mt-24 mb-36">
        <div className="text-center px-6 max-xl mx-auto pb-16">
          <h2 className="text-4xl">
            Your{" "}
            <span className="gradient-text gradient-text-ltr gradient-from-cyan gradient-to-blue">
              Solution
            </span>{" "}
            for...
          </h2>
        </div>
        {/* Change this grid as we add more use cases */}
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6 mx-6 max-w-5xl">
          {useCases.map((u, i) => (
            <a
              key={`use-case-anchor-${i}`}
              href={u.href}
              className={`bg-light-gray p-8 rounded-lg text-almost-black transition ease-in-out duration-200 hover:-translate-y-1`}
            >
              <div className="mb-6">
                <h3 className="text-2xl mb-6">
                  <img
                    className="inline-flex mr-2"
                    src={u.icon}
                    style={{ maxWidth: "28px" }}
                  />
                  {u.title}
                </h3>
              </div>
              <p>{u.description}</p>
            </a>
          ))}
        </div>
      </div>

      <FeatureCallouts
        heading={
          <>
            Build powerful functionality
            <br />
            <span className="gradient-text gradient-text-ltr gradient-from-cyan gradient-to-pink">
              without the overhead
            </span>
          </>
        }
        backgrounds="gray"
        features={[
          {
            topic: "Event Bus",
            title: "Send data and view full history",
            description:
              "Publish your events and view full logs of events including the payload, event schema, and what functions it triggered.",
            image: "/assets/screenshots/dashboard-events.png",
          },
          {
            topic: "Developer UX",
            title: "Intuitive Developer Tooling",
            description:
              "A CLI that gets out your way and makes the hard stuff easy. Create, test, and deploy functions in minutes.",
            image: "/assets/homepage/cli-3-commands.png",
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
        ]}
        cta={{
          href: "/sign-up?ref=homepage-features",
          text: "Get started building now →",
        }}
      />

      <GraphicCallout
        heading="Open Source"
        description="Inngest's core is open source, giving you piece of mind."
        image="/assets/screenshots/github-repo-inngest-top-left.png"
        cta={{
          href: "https://github.com/inngest/inngest?ref=inngest-homepage",
          text: "Star the repo",
        }}
      />

      {/* <SocialProof>
        <blockquote>
          “This is 100% the dev/prod parity that we’re lacking for queue-based
          systems.”
        </blockquote>
        <div className="attribution">
          <img src="/assets/team/dan-f-2022-02-18.jpg" />
          Developer A. - Staff Engineer at XYZ
        </div>
        </SocialProof> */}

      <DemoBlock
        headline="Inngest provides the tools for any automation"
        description="Skip the boilerplate and get right to the heart of the matter: writing code that helps your business achieve its goals."
      />

      <Footer />
    </div>
  );
}

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

const Demo = styled.div`
  position: fixed;
  top: 0;
  z-index: 10;
  left: 0;
  width: 100%;
  max-width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.4);

  > div {
    box-shadow: 0 0 60px rgba(0, 0, 0, 0.5);
  }
`;

const TextSlider = ({ strings = [] }) => {
  const [index, setIndex] = useState(0);

  const DELAY = 3000;

  useEffect(() => {
    const interval = setInterval(() => {
      setIndex((i) => (i + 1) % strings.length);
    }, DELAY);
    return () => clearInterval(interval);
  }, []);

  const classes = [
    "gradient-from-iris-60 gradient-to-cyan",
    "gradient-from-cyan gradient-to-pink",
    "gradient-from-pink gradient-to-orange",
    "gradient-from-orange gradient-to-red",
  ];

  return (
    <TextSliderContainer>
      <span style={{ position: "relative" }}>
        <TextSliderElements>
          {strings.map((s, i) => (
            <TextSliderString
              key={`string-${i}`}
              className={`gradient-text-ltr ${classes[i]} ${
                index === i
                  ? "active"
                  : index - 1 === i || (index === 0 && i === strings.length - 1)
                  ? "previous"
                  : "upcoming"
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
const TextSliderPlaceholder = styled.span`
  visibility: hidden;
  z-index: -10;
`;
const TextSliderElements = styled.span`
  position: relative;
  z-index: 1;
  top: 12%;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  white-space: nowrap;
  height: 100%;
  align-items: start;
  transition: all ease-out 200ms;
`;
const TextSliderString = styled.span`
  position: absolute;
  top: 0px;
  width: 600px;
  width: 100%;
  text-align: center;
  transition: all cubic-bezier(0.32, 0.8, 0.87, 0.85) 200ms;
  opacity: 0;

  &.active {
    opacity: 1;
  }
  &.previous {
    transform: translateX(-100%) translateY(-50%) scale(10%);
  }
  &.upcoming {
    transform: translateX(100%);
  }
`;
