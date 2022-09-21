import React, { ReactElement } from "react";
import { Zap, Code, Smile } from "react-feather";

import StepGrid, { Step } from "src/shared/StepGrid";

type HowItWorksProps = {
  cta?: {
    href: string;
    text: string;
  };
};

const steps: Step[] = [
  {
    icon: <Zap />,
    description: "Event trigger",
    action: "via SDK, Webhooks, Integrations",
  },
  {
    icon: <Code />,
    description: "Serverless functions",
    action: "JavaScript & TypeScript",
  },
  {
    icon: <Smile />,
    description: "Inngest Cloud",
    action: "Full history & metrics. 100% managed.",
  },
];
const HowItWorks = ({ cta }: HowItWorksProps) => {
  return (
    <div
      style={{ backgroundColor: "#f8f7fa" }}
      className="background-grid-texture"
    >
      <div className="container mx-auto max-w-5xl px-6 py-6">
        <div className="text-center px-6 max-xl mx-auto py-16">
          <h2 className="text-3xl sm:text-4xl font-normal">
            How Inngest Works
          </h2>
        </div>
        <div className="grid md:grid-cols-3 gap-6 md:gap-16 items-start py-6">
          <div>
            <h3 className="text-lg mb-2">Select your event trigger</h3>
            <p className="my-0">
              Send events directly from your application with{" "}
              <a href="/docs/events?ref=how-it-works">our SDK</a>, configure a
              webhook, connect an integration.
              {/* TODO - link to page when it exists */}
            </p>
          </div>
          <div>
            <h3 className="text-lg mb-2">Write a function</h3>
            <p className="my-0">
              Use our JavaScript & TypeScript SDK to{" "}
              <a href="/docs/functions?ref=how-it-works">
                create a background job
              </a>{" "}
              triggered by an event or{" "}
              <a href="/docs/functions?ref=how-it-works#writing-a-scheduled-function">
                on a schedule
              </a>
              .
            </p>
          </div>
          <div>
            <h3 className="text-lg mb-2">Deploy & Relax</h3>
            <p className="my-0">
              Deploy your functions to your existing platform. Inngest runs your
              code every time a matching event trigger is received. <br />
              <em>No infra to manage.</em>
            </p>
          </div>
        </div>
        <StepGrid steps={steps} />
        {cta && (
          <div className="text-center px-6 max-xl mx-auto py-16">
            <p className="text-2xl font-normal italic">
              <a href={cta.href}>{cta.text}</a>
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default HowItWorks;
