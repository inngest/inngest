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
    action: "Webhooks, Integrations, API Keys",
  },
  {
    icon: <Code />,
    description: "Business Logic",
    action: "Any programming language",
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
          <h2 className="text-4xl font-normal	">How Inngest Works</h2>
        </div>
        <div className="grid md:grid-cols-3 gap-6 md:gap-16 items-start py-6">
          <div>
            <h3 className="text-lg mb-2">Select your event trigger</h3>
            <p className="my-0">
              Configure a webhook, connect an integration, or send data directly
              from your application code with an API Key.
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
              Deploy your function in seconds. Inngest runs your code every time
              a matching event trigger is received. <br />
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
