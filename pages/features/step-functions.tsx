import { useState } from "react";

import Button from "src/shared/Button";
import CodeWindow from "src/shared/CodeWindow";
import Footer from "src/shared/footer";
import CheckboxUnchecked from "src/shared/Icons/CheckboxUnchecked";
import CheckRounded from "src/shared/Icons/CheckRounded";
import Discord from "src/shared/Icons/Discord";
import Nav from "src/shared/nav";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Step Functions",
        description:
          "Build complex serverless workflows with delays, conditional steps and coordinate between events",
      },
    },
  };
}

export const BETA_TYPEFORM_URL = "https://8qe8m10yfz6.typeform.com/to/F1aj8vLl";

const codesnippets = {
  main: `
    createStepFunction("Post-signup", "user/created",
      function ({ event, tools }) {
        // Send the user an email
        tools.run("Send an email", async () => {
          await sendEmail({
            email: event.user.email,
            template: "welcome",
          });
        });

        // Wait for the user to create an order, by waiting and
        // matching on another event
        const order = tools.waitForEvent("order/created", {
          match: ["data.user.id"],
          timeout: "24h"
        })

        if (order === null) {
          // User didn't create an order;  send them a activation email
          tools.run("Send activation", async () => {
            // Some code here
          })
        }
      }
);
  `,
  eventCoordination: `
    const seenEvent = tools.waitForEvent("app/user.seen", {
      match: ["data.email"],
      timeout: "2d",
    });
  `,
  delay: `
    tools.run("Do something", () => { ... })

    // Pause the function and resume in 2d
    tools.sleep("2d")

    tools.run("Do something later", () => { ... })
  `,
  conditional: `
    const result = tools.run("Send customer outreach", () => { ... })

    if (result.requiresFollowUp) {
      const result = tools.run("Add task to Linear", () => { ... })
    } else {
      const result = tools.run("Mark task complete", () => { ... })
    }
  `,
};

export default function FeaturesSDK() {
  return (
    <div>
      <Nav sticky={true} />

      {/* Background styles */}
      {/* <div className="bg-light-gray background-grid-texture bg-dark-rainbow-gradient"> */}
      <div className="bg-dark-rainbow-gradient">
        {/* Content layout */}
        <div className="-mt-6 mx-auto pt-24 pb-12 px-10 lg:px-4 max-w-3xl">
          <header className="pb-12 text-center text-white">
            <span className="px-4 py-1 rounded-full text-xs uppercase bg-black/50 border border-black">
              STEP FUNCTIONS BETA
            </span>
            <h1 className="my-6 text-6xl gradient-spotlight font-normal tracking-tighter">
              Step Functions
            </h1>
            <p className="mt-8 mx-auto md:max-w-xl">
              Build multi-step serverless workflows with delays, conditional
              logic and coordinate between events. Ship complex functionality in
              a fraction of time.
            </p>
          </header>
          <CodeWindow
            className="mx-auto max-w-xl shadow-xl"
            filename={`inngest/smartOnboardingDripCampaign`}
            snippet={codesnippets.main}
          />
          <div className="mt-10 flex flex-wrap gap-6 justify-center items-center">
            <Button href={"#request-beta-access"} kind="primary" size="medium">
              Request Beta Access
            </Button>
          </div>
        </div>
      </div>

      {/* Background styles */}
      <section>
        {/* Content layout */}
        <div className="mx-auto my-28 px-6 lg:px-4 max-w-4xl">
          <header className="mb-12 text-center">
            <h2 className="text-4xl">Write code, not config</h2>
          </header>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 md:gap-y-12">
            <div>
              <CodeWindow
                className="shadow-xl hover:transform-iso-opposite"
                filename={`inngest/eventCoordination.ts`}
                snippet={codesnippets.eventCoordination}
              />
            </div>
            <div>
              <h3 className="text-2xl">Coordinate between events</h3>
              <p className="max-w-sm my-6">
                Create dynamic workflows based on multiple events without having
                to keep state and poll for updates.
              </p>
              {/* <p>
                <a href="/docs/events?ref=features-sdk">Read the docs →</a>
              </p> */}
            </div>

            <div>
              <CodeWindow
                className="shadow-xl hover:transform-iso-opposite"
                filename={`inngest/delayedCode.ts`}
                snippet={codesnippets.delay}
              />
            </div>
            <div>
              <h3 className="text-2xl">
                Delay for hours <em>or days</em>
              </h3>
              <p className="max-w-sm my-6">
                Add delays within your step function enabling you to build jobs
                that pause and resume over multiple days.
              </p>
              {/* <p>
                <a href="/docs/functions?ref=features-sdk">Learn how →</a>
              </p> */}
            </div>

            <div>
              <CodeWindow
                className="shadow-xl hover:transform-iso-opposite"
                filename={`inngest/conditionalSteps.ts`}
                snippet={codesnippets.conditional}
              />
            </div>
            <div>
              <h3 className="text-2xl">Conditionally run steps</h3>
              <p className="max-w-sm my-6">
                Use the result of steps to determine <em>if</em> or{" "}
                <em>what</em> step should run next.
              </p>
              {/* <p>
                <a href="/docs/deploy?ref=features-sdk">Get more info →</a>
              </p> */}
            </div>
          </div>
        </div>
      </section>

      {/* Background styles */}
      <section>
        {/* Content layout */}
        {/* Less bottom margin due to next section padding for anchor */}
        <div className="mx-auto mt-28 px-6 lg:px-4 max-w-4xl">
          <header className="mb-12 text-center">
            <h2 className="text-4xl">
              Combine to create{" "}
              <span className="gradient-text gradient-text-ltr gradient-from-pink gradient-to-orange">
                awesome
              </span>{" "}
              things
            </h2>
          </header>
          <div className="grid md:grid-cols-3 gap-6 md:gap-16 items-start py-6">
            {[
              {
                title: "Automate user journeys",
                description:
                  "Build cross-channel onboarding, drip, re-activation campaigns to your team's custom needs.",
              },
              {
                title: "Create internal tooling",
                description:
                  "Use events from Zendesk, Retool, Intercom to build automate tedious steps for your team.",
              },
              {
                title: "Build data pipelines",
                description:
                  "Enrich, process, and forward event data into any destination imaginable.",
              },
            ].map((u) => (
              <div>
                <h3 className="mb-2 text-lg">{u.title}</h3>
                <p>{u.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Background styles */}
      <div id="request-beta-access">
        {/* Content layout */}
        <div className="mx-auto py-28 px-6 lg:px-4 max-w-4xl">
          <header className="mt-24 mb-12 text-center">
            <h2 className="text-4xl">
              Join the{" "}
              <span className="gradient-text gradient-text-ltr">
                Step Functions Beta
              </span>
            </h2>
            <p className="mt-8 mx-auto max-w-xl">
              We're testing step functions as a closed beta. If you want get
              early access and provide feedback to help shape the future of
              Inngest, get in touch with the team:
            </p>
          </header>
          <div className="my-10 flex flex-col sm:flex-row flex-wrap gap-6 justify-center items-center">
            <Button href="/contact" kind="outline" style={{ margin: 0 }}>
              Send us an email
            </Button>
            <Button
              href="https://www.inngest.com/discord"
              kind="outline"
              style={{ margin: 0 }}
            >
              <Discord /> Reach out on our Discord
            </Button>
          </div>
        </div>
      </div>

      <Footer />
    </div>
  );
}
