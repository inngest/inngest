import React, { useEffect, useState } from "react";
import styled from "@emotion/styled";

import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Button from "src/shared/Button";
import CheckRounded from "src/shared/Icons/CheckRounded";
import HowItWorks from "src/shared/HowItWorks";
import FeatureCallouts from "src/shared/FeatureCallouts";
import DemoBlock from "src/shared/DemoBlock";
import GraphicCallout from "src/shared/GraphicCallout";
import CodeWindow from "src/shared/CodeWindow";
import Discord from "src/shared/Icons/Discord";

import {
  Hero as SDKHero,
  codesnippets,
  worksWithBrands,
  BETA_TYPEFORM_URL,
} from "./features/sdk";
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
        title: "Write functions, Send Events",
        description:
          "Inngest is a developer platform for building, testing and deploying code that runs in response to events or on a schedule — without spending any time on infrastructure.",
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

      {/* Hero copied from sdk page */}
      <div>
        {/* Content layout */}
        <div className="mx-auto my-12 px-10 lg:px-16 max-w-5xl grid grid-cols-1 lg:grid-cols-2 gap-8">
          <header className="lg:my-24 mt-8">
            <h1
              className="mt-2 mb-6 text-3xl sm:text-5xl leading-tight sm:overflow-hidden"
              style={{ lineHeight: "1.08" }}
            >
              Build
              <br />
              <TextSlider
                strings={[
                  "Background Jobs",
                  "Webhooks",
                  "Internal Tools",
                  "User Journeys",
                ]}
              />
              <br />
              in Minutes
            </h1>
            <p>
              Inngest is a developer platform for building, testing and
              deploying code that runs in response to events or on a schedule —
              without spending any time on infrastructure.
            </p>
            <div className="mt-10 flex flex-wrap gap-6 justify-start items-center">
              <Button
                href="/sign-up?ref=homepage-hero"
                kind="primary"
                size="medium"
              >
                Get started for free →
              </Button>
              <Button
                href="/docs?ref=homepage-hero"
                kind="outline"
                size="medium"
                style={{ margin: 0 }}
              >
                Read the docs
              </Button>
            </div>
          </header>
          <div className="mt-6 lg:mt-12 mx-auto lg:mx-6 max-w-full md:max-w-lg flex flex-col">
            <CodeWindow
              className="transform-iso shadow-xl relative z-10"
              filename={`myGreatFunction.${ext}`}
              snippet={codesnippets[language].function}
            />
            <CodeWindow
              className="mt-6 transform-iso-opposite shadow-xl relative"
              filename={`api/someEndpoint.${ext}`}
              snippet={codesnippets[language].sendEventShort}
            />
          </div>
        </div>
      </div>

      <div className="mx-auto max-w-5xl mb-24 mt-20 sm:mt-0">
        <div className="text-center px-6 max-w-4xl mx-auto">
          <h2 className="text-2xl mb-6">Works with</h2>
          <div className="mt-4 flex flex-wrap items-center justify-center gap-8 h-8 sm:h-10">
            {worksWithBrands.map((b) => (
              <a
                key={`brand-${b.brand}`}
                href={`${b.docs}?ref=homepage-works-with`}
                className="block bulge"
                style={{ height: b.height }}
              >
                {/* href={`${b.docs}?ref=homepage-works-with`} TODO: Update this for each doc */}
                <img
                  key={b.brand}
                  src={b.logo}
                  alt={`${b.brand}'s logo`}
                  className="h-full"
                />
              </a>
            ))}
            {/* TODO: Should this also just have "JavaScript/TypeScript" logos? */}
          </div>
        </div>
      </div>

      <HowItWorks />

      <UseCases options={useCaseList} />

      {/* Background styles */}
      <div className="">
        {/* Content layout */}
        <div className="mx-auto my-28 px-10 lg:px-4 max-w-4xl">
          <header className="my-24 text-center">
            <h2 className="text-3xl md:text-4xl">
              The Complete Platform For{" "}
              <span className="sm:whitespace-nowrap gradient-text gradient-text-ltr gradient-from-pink gradient-to-orange">
                Everything Async
              </span>
            </h2>
            <p className="mt-8">
              Our serverless solution provides everything you need to
              effortlessly
              <br />
              build and manage every type of asynchronous and event-driven job.
            </p>
          </header>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 lg: gap-y-12">
            <div className="md:h-48 flex flex-col justify-center items-center">
              <div
                className="w-72 relative grid grid-cols-8 gap-0 transform-iso-opposite rounded-lg border-4 border-transparent"
                style={{
                  maxWidth: "340px",
                  background:
                    "linear-gradient(#fff, #fff) padding-box, linear-gradient(to right, #5D5FEF, #EF5F5D) border-box",
                }}
              >
                <div className="absolute right-1" style={{ top: "-3rem" }}>
                  <CheckRounded fill="#5D5FEF" size="5rem" />
                </div>
                {[1, 2, 3, 4, 5, 6, 7, 8].map((n) => (
                  <div
                    key={n}
                    className={`h-12 bg-white border-slate-200 ${
                      n !== 8 ? "border-r-4" : "rounded-r-md"
                    } ${n === 1 ? "rounded-l-md" : ""}
                    `}
                    style={{
                      animation: `queue-message-flash 4s infinite ${n / 2}s`,
                    }}
                  >
                    &nbsp;
                  </div>
                ))}
              </div>
            </div>
            <div>
              <h3 className="text-2xl">No infrastructure to manage</h3>
              <p className="my-6">
                Inngest is serverless, requiring absolutely no infra for you to
                manage. No queues, event bus, or logging to configure.
              </p>
            </div>

            <div className="md:h-48 flex flex-col justify-center items-center">
              <img
                src="/assets/homepage/admin-ui-screenshot.png"
                className="rounded-sm transform-iso-opposite"
                style={{ maxWidth: "340px" }}
              />
            </div>
            <div>
              <h3 className="text-2xl">
                A real-time dashboard keeps everyone in the loop
              </h3>
              <p className="my-6">
                The Inngest Cloud Dashboard brings full transparency to all your
                asynchronous jobs, so you can stay on top of performance,
                throughput, and more, without needing to dig through logs.
              </p>
            </div>

            <div className="md:h-48 flex flex-col justify-center items-center">
              <div
                className="transform-iso-opposite flex flex-col gap-1"
                style={{ boxShadow: "none" }}
              >
                {[
                  "Automatic Retries",
                  "Event Replay",
                  "Versioning",
                  "Idempotency",
                ].map((s) => (
                  <div key={s} className="flex flex-row items-center gap-2">
                    <CheckRounded fill="#5D5FEF" size="1.6rem" shadow={false} />{" "}
                    {s}
                  </div>
                ))}
              </div>
            </div>
            <div>
              <h3 className="text-2xl">
                Event-driven, as easy as just sending events!
              </h3>
              <p className="my-6">
                We built all the hard stuff so you don't have to: idempotency,
                throttling, backoff, retries,{" "}
                <a href="/blog/introducing-cli-replays?ref=homepage">replays</a>
                , job versioning, and so much more. With Inngest, you just write
                your code and we take care of the rest.
              </p>
            </div>
          </div>
          <div className="my-10 flex justify-center">
            <Button
              href="/docs/functions?ref=homepage-platform"
              kind="outlinePrimary"
            >
              Try the SDK →
            </Button>
          </div>
        </div>
      </div>

      <FeatureCallouts
        heading={
          <>
            Built for{" "}
            <span className="gradient-text gradient-text-ltr gradient-from-cyan gradient-to-pink">
              Builders
            </span>
          </>
        }
        backgrounds="gray"
        features={[
          {
            topic: "Supported frameworks",
            title: "Fits your existing project",
            description: (
              <>
                Inngest fits right into your current project and workflow so you
                can focus on shipping.
                <br />
                <br />
                Guides:{" "}
                <a href="/docs/frameworks/nextjs?ref=homepage">Next.js</a>{" "}
                &middot;{" "}
                <a href="/docs/frameworks/express?ref=homepage">Express</a>
              </>
            ),
            image: (
              <>
                <div className="grid grid-rows-2 grid-cols-2 items-center	gap-8 max-h">
                  {worksWithBrands
                    .filter((b) => b.type === "framework")
                    .concat([
                      {
                        docs: "",
                        brand: "TypeScript",
                        logo: "/assets/brand-logos/typescript.svg",
                        height: "100%",
                        type: "language",
                      },
                      {
                        docs: "",
                        brand: "JavaScript",
                        logo: "/assets/brand-logos/javascript.svg",
                        height: "100%",
                        type: "language",
                      },
                    ])
                    .map((b) => (
                      <img
                        src={b.logo}
                        className="max-h-20"
                        style={{ height: `calc(${b.height} /3 }` }}
                      />
                    ))}
                </div>
              </>
            ),
          },
          {
            topic: "Easy to learn",
            title: "Implement in seconds",
            description: (
              <>
                <code className="text-xs text-color-secondary">
                  npm install inngest
                </code>{" "}
                and you're on your way to writing background jobs or scheduled
                functions.
                <br />
                <br />
                <a href="/docs/functions?ref=homepage">
                  Learn how to get started →
                </a>
              </>
            ),
            image: (
              <div className="flex flex-col justify-around w-full h-full">
                <CodeWindow
                  theme="dark"
                  snippet="$ npm install inngest"
                  type="terminal"
                  className="w-52 relative sm:z-20 self-center sm:self-start shadow-md"
                />
                <CodeWindow
                  className="w-full min-w-min	z-40 sm:z-10 mt-1 sm:mt-0 sm:w-80 self-center sm:self-end shadow-md "
                  filename={`function.js`}
                  snippet={`
          import { createFunction } from "inngest"

          export default createFunction(
            "My function",
            "demo/some.event",
            async ({ event }) => {
              // your business logic
              return "awesome"
            }
          )
          `}
                />
              </div>
            ),
          },
          {
            topic: "No infra",
            title: "Zero configuration or extra infra to set up",
            description: (
              <>
                Your code tells Inngest how it should be run so there is no
                extra yaml or json config to write. You can deploy functions to
                your existing production setup or to Inngest Cloud
                <br />
                <br />
                Deploy to: <a href="/docs/deploy/vercel?ref=homepage">
                  Vercel
                </a>{" "}
                | <a href="/docs/deploy/netlify?ref=homepage">Netlify</a> |{" "}
                <a href="/docs/deploy/cloudflare?ref=homepage">
                  Cloudflare&nbsp;Pages
                </a>{" "}
                |{" "}
                <a href="/docs/deploy/inngest-cloud?ref=homepage">
                  Inngest&nbsp;Cloud&nbsp;(Coming soon)
                </a>{" "}
                |{" "}
                <a href="/docs/deploy/aws-lambda?ref=homepage">
                  AWS&nbsp;Lambda&nbsp;(Waitlist)
                </a>
              </>
            ),
            image: (
              <>
                <p
                  className="text-lg sm:text-2xl md:text-6xl font-bold drop-shadow-xl bg-clip-text text-transparent bg-gradient-to-t from-amber-400 via-orange-400 to-red-500"
                  style={{ transform: "skewY(-6deg)" }}
                >
                  🔥 Serverless
                </p>
              </>
            ),
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

      {/* Background styles */}
      <div className="">
        {/* Content layout */}
        <div className="mx-auto my-28 px-6 lg:px-4 max-w-4xl">
          <header className="mt-24 mb-12 text-center">
            <h2 className="text-4xl">
              Join the{" "}
              <span className="gradient-text gradient-text-ltr">
                Inngest Community
              </span>
            </h2>
            <p className="mt-8 mx-auto max-w-xl">
              Join our Discord community to share feedback, get updates, and
              have a direct line to shaping the future of the SDK!
            </p>
          </header>
          <div className="my-10 flex flex-col sm:flex-row gap-6 justify-center items-center">
            <Button
              href="https://www.inngest.com/discord"
              kind="outline"
              style={{ margin: 0 }}
            >
              <Discord /> Join our community on Discord
            </Button>
          </div>
        </div>
      </div>

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

      {/* <DemoBlock
        headline="Inngest provides the tools for any automation"
        description="Skip the boilerplate and get right to the heart of the matter: writing code that helps your business achieve its goals."
      /> */}

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
              align="left"
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

  @media (max-width: 640px) {
    top: auto;
    display: inline-flex;
  }
`;
const TextSliderString = styled.span<{ align: "center" | "left" }>`
  position: absolute;
  top: 0px;
  width: 600px;
  width: 100%;
  text-align: ${({ align }) => align};
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

  // disable the animation and stack items on mobile - Match tailwind breakpoint
  @media (max-width: 640px) {
    position: inherit;
    white-space: normal;
    opacity: 1;
    &.previous,
    &.upcoming {
      transform: none;
    }
  }
`;

const useCaseList = [
  {
    title: "Background jobs",
    href: "/uses/serverless-node-background-jobs?ref=homepage-use-cases",
    description: [
      {
        heading: "Out of the critical path",
        description:
          "Ensure your API is fast by running your code, asynchronously, in the background.",
      },
      {
        heading: "No queues or workers required",
        description:
          "Serverless background jobs mean you don’t need to set up queues or long-running workers.",
      },
    ],
    graphic: (
      <>
        <CodeWindow
          className="shadow-md"
          filename={`sendConfirmationSMS.js`}
          snippet={`
          import { createFunction } from "inngest"
          import { sendSMS } from "../twilioUtils"

          export default createFunction(
            "Send confirmation SMS",
            "app/request.confirmed",
            async ({ event }) => {
              const result = await sendSMS({
                to: event.user.phone,
                message: "Your request has been confirmed!",
              })
              return {
                status: result.ok ? 200 : 500,
                body: \`SMS Sent (Message SID: \${result.sid})\`
              }
            }
          )
          `}
        />
      </>
    ),
  },
  {
    title: "Scheduled jobs",
    href: "/uses/serverless-node-background-jobs?ref=homepage-use-cases", // TODO - all links
    description: [
      {
        heading: "Serverless cron jobs",
        description:
          "Run your function on a schedule to repeat hourly, daily, weekly or whatever you need.",
      },
      {
        heading: "No workarounds needed",
        description:
          "Tell Inngest when to run it and we'll take care of the rest",
      },
    ],
    graphic: (
      <>
        <CodeWindow
          className="shadow-md"
          filename={`sendWeeklyDigest.js`}
          snippet={`
          import { createScheduledFunction } from "inngest"
          import { sendWeeklyDigestEmails } from "../emails"

          export default createScheduledFunction(
            "Send Weekly Digest",
            "0 9 * * MON",
            sendWeeklyDigestEmails
          )
          `}
        />
      </>
    ),
  },
  {
    title: "Webhooks",
    href: "/uses/serverless-node-background-jobs?ref=homepage-use-cases",
    description: [
      {
        heading: "Build reliable webhooks",
        description:
          "Inngest acts as a layer which can handle webhook events and that run your functions automatically.",
      },
      {
        heading: "Full observability",
        description:
          "The Inngest Cloud dashboard gives your complete observability into what event payloads were received and how your functions ran.",
      },
    ],
    graphic: (
      <>
        <CodeWindow
          className="shadow-md"
          filename={`handleFailedPayments.js`}
          snippet={`
          import { createFunction } from "inngest"
          import {
            findAccountByCustomerId, downgradeAccount
          } from "../accounts"
          import { sendFailedPaymentEmail } from "../emails"

          export default createFunction(
            "Handle failed payments",
            "stripe/charge.failed",
            async ({ event }) => {
              const account = await = findAccountByCustomerId(event.user.stripe_customer_id)
              await sendFailedPaymentEmail(account.email)
              await downgradeAccount(account.id)
              return { message: "success" }
            }
          )
          `}
        />
      </>
    ),
  },
  {
    title: "Internal Tools",
    href: "/uses/internal-tools?ref=homepage-use-cases",
    description: [
      {
        heading: "Trigger scripts on-demand",
        description:
          "Easily run necessary scripts on-demand triggered from tools like Retool or your own internal admin.",
      },
      {
        heading: "Run code with events from anywhere",
        description:
          "Slack or Stripe webhook events can trigger your code to run based off things like refunds or Slackbot interactions.",
      },
    ],
    graphic: (
      <>
        <CodeWindow
          className="shadow-md"
          filename={`runUserDataBackfill.js`}
          snippet={`
          import { createFunction } from "inngest"
          import { runBackfillForUser } from "../scripts"

          export default createFunction(
            "Run user data backfill",
            "retool/backfill.requested",
            async ({ event }) => {
              const result = await runBackfillForUser(event.data.user_id)
              return {
                status: result.ok ? 200 : 500,
                body: \`Ran backfill for user \${event.data.user_id}\`
              }
            }
          )
          `}
        />
      </>
    ),
  },
  {
    title: "User Journey Automation",
    href: "/uses/user-journey-automation?ref=homepage-use-cases",
    description: [
      {
        heading: "User-behavior driven",
        description:
          "Build out user-behavior driven flows for your product that are triggered by events sent from your app or third party integrations like drip email campaigns, re-activation campaigns, or reminders.",
      },
      {
        heading: "Step functions (Coming soon!)",
        description:
          "Add delays, connect multiple events, and build multi-step workflows to create amazing personalized experiences for your users.",
      },
    ],
    graphic: (
      <>
        <CodeWindow
          className="shadow-md"
          filename={`userOnboardingCampaign.js`}
          snippet={`
          import { createStepFunction } from "inngest"

          export default createStepFunction(
            "User onboarding campaign",
            "app/user.signup",
            /*
              Coming soon!
              Join the feedback group on Discord
            */
          )
          `}
        />
      </>
    ),
  },
  {
    title: "Event-driven Systems",
    href: "/uses/user-journey-automation?ref=homepage-use-cases",
    description: [
      {
        heading: "Design around events",
        description:
          "Developers can send and subscribe to a variety of internal and external events, creating complex event-driven architectures without worrying about infrastructure and boilerplate.",
      },
      {
        heading: "Auto-generated event schemas",
        description:
          "Events are parsed and schemas are generated and versioned automatically as you send events giving more oversight to the events that power your application.",
      },
    ],
    graphic: (
      <>
        <CodeWindow
          className="shadow-md"
          filename={`eventDriven.js`}
          snippet={`
          import { createFunction } from "inngest"

          export const handleApptRequested = createFunction("...",
            "appointment.requested", // ...
          )
          export const handleApptScheduled = createFunction("...",
            "appointment.scheduled", // ...
          )
          export const handleApptConfirmed = createFunction("...",
            "appointment.confirmed", // ...
          )
          export const handleApptCancelled = createFunction("...",
            "appointment.cancelled", // ...
          )
          `}
        />
      </>
    ),
  },
];

const UseCases = ({ options }) => {
  const [selected, setSelected] = useState(options[0]);
  return (
    <section>
      {/* Content layout */}
      <div className="mx-auto my-16 py-10 px-6 md:px-16 max-w-5xl bg-violet-100 rounded-lg">
        <h2 className="text-2xl sm:text-4xl mt-2 mb-2">Get things shipped</h2>
        <p className="text-sm text-color-secondary">
          Inngest's platform enables you to ship features quickly without the
          overhead.
        </p>
        <div className="my-5">
          {options.map((o) => (
            <button
              className={`py-1 px-3 mr-2 my-1 text-sm font-medium rounded-md ${
                selected.title === o.title
                  ? "bg-violet-500 text-white drop-shadow"
                  : "hover:bg-slate-100"
              }`}
              onClick={() => setSelected(o)}
            >
              {o.title}
            </button>
          ))}
        </div>
        <div className="mt-10 grid grid-cols-1 lg:grid-cols-10 gap-12">
          <div className="lg:col-span-4">
            {selected?.description.map((d) => (
              <>
                <h3 className="text-base mb-2">{d.heading}</h3>
                <p className="text-sm mb-4">{d.description}</p>
              </>
            ))}
          </div>
          <div className="lg:col-span-6 h-80">{selected?.graphic}</div>
        </div>
      </div>
    </section>
  );
};
