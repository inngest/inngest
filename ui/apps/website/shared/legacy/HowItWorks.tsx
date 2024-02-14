import React, { ReactElement } from 'react';
import { Code, Smile, Zap } from 'react-feather';

import StepGrid, { Step } from './StepGrid';

type HowItWorksProps = {
  cta?: {
    href: string;
    text: string;
  };
};

const steps: Step[] = [
  {
    icon: <Zap />,
    description: 'Event trigger',
    action: 'via SDK, Webhooks, Integrations',
  },
  {
    icon: <Code />,
    description: 'Serverless functions',
    action: 'JavaScript & TypeScript',
  },
  {
    icon: <Smile />,
    description: 'Inngest Cloud',
    action: 'Full history & metrics. 100% managed.',
  },
];
const HowItWorks = ({ cta }: HowItWorksProps) => {
  return (
    <div style={{ backgroundColor: '#f8f7fa' }} className="background-grid-texture">
      <div className="container mx-auto max-w-5xl px-6 py-6">
        <div className="max-xl mx-auto px-6 py-16 text-center">
          <h2 className="text-3xl font-normal sm:text-4xl">How Inngest Works</h2>
        </div>
        <div className="grid items-start gap-6 py-6 md:grid-cols-3 md:gap-16">
          <div>
            <h3 className="mb-2 text-lg">Select your event trigger</h3>
            <p className="my-0">
              Send events directly from your application with{' '}
              <a href="/docs/events?ref=how-it-works">our SDK</a>, configure a webhook, connect an
              integration.
              {/* TODO - link to page when it exists */}
            </p>
          </div>
          <div>
            <h3 className="mb-2 text-lg">Write a function</h3>
            <p className="my-0">
              Use our JavaScript & TypeScript SDK to{' '}
              <a href="/docs/functions?ref=how-it-works">create a background job</a> triggered by an
              event or{' '}
              <a href="/docs/functions?ref=how-it-works#writing-a-scheduled-function">
                on a schedule
              </a>
              .
            </p>
          </div>
          <div>
            <h3 className="mb-2 text-lg">Deploy & Relax</h3>
            <p className="my-0">
              Deploy your functions to your existing platform. Inngest runs your code every time a
              matching event trigger is received. <br />
              <em>No infra to manage.</em>
            </p>
          </div>
        </div>
        <StepGrid steps={steps} />
        {cta && (
          <div className="max-xl mx-auto px-6 py-16 text-center">
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
