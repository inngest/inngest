import React, { useState } from 'react';
import styled from '@emotion/styled';
import { Activity, Code, Eye } from 'react-feather';
import Button from 'src/shared/legacy/Button';
import Footer from 'src/shared/legacy/Footer';
import GraphicCallout from 'src/shared/legacy/GraphicCallout';
import Nav from 'src/shared/legacy/nav';

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: 'Internal tools, solved in seconds',
        description:
          'Build and deploy internal apps using any language, with full audit trails, human in the loop tasks, and automated flows. Build and ship using the most advanced tooling platform available.',
      },
    },
  };
}

export default function Template() {
  const [demo, setDemo] = useState(false);

  return (
    <div>
      <Nav sticky={true} nodemo />
      <div className="container mx-auto flex flex-row py-32">
        <div className="mx-auto max-w-4xl px-6 text-center">
          <h1>Internal tools, solved in seconds.</h1>
          <p className="subheading pt-6">
            Inngest allows you to <b>build and deploy internal apps using any language</b>, with
            full audit trails, human in the loop tasks, and automated flows. Build and ship using
            the most advanced tooling platform available.
          </p>
          <div className="flex flex-row justify-center pt-12">
            <Button kind="primary" href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=tools`}>
              Sign up
            </Button>
            <Button kind="outline" href="/contact?ref=tools">
              Get a demo
            </Button>
          </div>
        </div>
      </div>

      <Highlights>
        <div className="container mx-auto max-w-5xl py-24">
          <div className="max-w-2xl">
            <h2>
              <span className="gradient-text">Take control</span> of your tooling
            </h2>
            <p className="mb-0 pt-4" style={{ fontSize: '1rem' }}>
              Develop, test, and deploy internal apps and tools using a single CLI and a standard,
              familiar SDLC with <b>local testing, Git support, CI/CD</b>.
            </p>
            <p className="mt-2 pb-16" style={{ fontSize: '1rem' }}>
              Inngest allows you to turn serverless step functions into complex internal tools to
              automate any process.
            </p>
          </div>
          <div className="grid grid-cols-3 gap-16">
            <div>
              <Code />
              <h3 className="py-2">World-class dev UX</h3>
              <p>
                Locally build, test, and deploy functions using a single CLI, then integrate CI/CD
                via Git as expected from modern tooling. Plus, with full app versioning, immediate
                rollback, and historic replay of production data your team can manage internal tools
                as a first-class feature.
              </p>
            </div>
            <div>
              <Eye />
              <h3 className="py-2">Fully audited</h3>
              <p>
                Inngest stores every run of your function, who ran it, and, if the function relates
                to customers, the customer of record &mdash; allowing you to see every function run
                by employee, customer, or team.
              </p>
            </div>
            <div>
              <Activity />
              <h3 className="py-2">One platform for everything</h3>
              <p>
                Run internal apps manually or automate them with schedules or incoming triggers,
                allowing you to create complex apps which run whenever things happen in your
                product.
              </p>
            </div>
          </div>
        </div>
      </Highlights>

      <div className="container mx-auto flex justify-center py-24">
        <div className="align-center flex flex-col justify-center pr-24 text-center">
          <p>
            <i>“Sooooo much easier than AWS”</i>
          </p>
          <small>Between</small>
        </div>
        <div>
          <Button kind="primary" href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=tools`}>
            Start building today
          </Button>
        </div>
      </div>

      <GraphicCallout
        heading="Trigger your code directly from Retool"
        description="See how you can easily run existing code and scripts right from Retool with the power and flexibility of Inngest"
        image="/assets/use-cases/guide-retool-inngest.png"
        cta={{
          href: '/docs/guides/trigger-your-code-from-retool?ref=user-journey-graphic-callout',
          text: 'Read the guide',
        }}
        style={{
          backgroundImage:
            'linear-gradient(135deg, rgba(171, 220, 255, 0.5) 0%, rgba(3, 150, 255, 0.5) 100%)',
        }}
      />

      <div className="container mx-auto pt-6 text-center">
        <h2 className="mx-auto max-w-lg pb-6">
          Connect to anything,
          <br />
          automate everything
        </h2>
        <p className="mx-auto max-w-2xl pb-24">
          Craft step functions which connect multiple systems and database to automate internal
          processes. Automate processes with triggers driven by external systems.
        </p>

        <div className="relative mx-auto aspect-video max-w-4xl">
          <img
            src="/assets/escalation.jpg"
            alt="Dashboard"
            className="rounded drop-shadow-2xl"
            style={{ boxShadow: '0 0 40px rgba(0, 0, 0, 0.3)' }}
          />

          <Logos>
            <Slack className="mx-auto rounded">
              <img src="/assets/ui-assets/source-logos/slack.svg" width="96" />
              <img src="/assets/ui-assets/approve.png" className="approve" />
            </Slack>
            <div className="flex py-2">
              <Stripe className="mr-2 rounded">
                <img src="/assets/ui-assets/source-logos/stripe-color.svg" />
              </Stripe>
              <Mailchimp className="rounded">
                <img src="/assets/ui-assets/source-logos/mailchimp-black.png" />
              </Mailchimp>
            </div>
          </Logos>
        </div>

        <div className="flex justify-center pt-12">
          <Button kind="primary" href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=tools`}>
            Sign up
          </Button>
          <Button kind="outline" href="/docs?ref=tools">
            Read the docs
          </Button>
        </div>
      </div>

      <Quote className="container mx-auto max-w-xl space-y-3 pt-24 text-center">
        <q>This is 100% the dev/prod parity that we’re lacking</q>
        <p>Staff Engineer at Buffer</p>
      </Quote>

      <Footer />

      {demo && (
        <Demo
          className="flex items-center justify-center"
          onClick={() => {
            setDemo(false);
          }}
        >
          <div className="container mx-auto flex aspect-video max-w-2xl">
            <iframe
              src="https://www.youtube.com/embed/qVXzYBcJmGU?autoplay=1"
              title="Inngest Product Demo"
              frameBorder="0"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
              allowFullScreen
              className="flex-1"
            ></iframe>
          </div>
        </Demo>
      )}
    </div>
  );
}

const Highlights = styled.div`
  background: var(--bg-color-d);
  color: #fff;

  h3 {
    font-size: 1.25rem;
  }
`;

const Icon = styled.div`
  box-shadow: 0 0 40px rgba(0, 0, 0, 0.2);
  background: #fff;
`;

const Logos = styled.div`
  display: block;
  position: absolute;
  top: -30px;
  left: 50%;
`;

const Slack = styled(Icon)`
  display: flex;
  align-items: center;
  width: fit-content;

  .approve {
    height: 52px;
    margin: 0 12px 0 -8px;
  }
`;

const Stripe = styled(Icon)`
  background: #fff;
  padding: 1rem;

  img {
    height: 30px;
  }
`;

const Mailchimp = styled(Icon)`
  background: #fff;
  padding: 1rem;

  img {
    height: 30px;
  }
`;

const Quote = styled.div`
  q {
    font-size: 1.8rem;
    font-weight: bold;
    font-style: italic;
    line-height: 1.2;
    color: var(--color-gray-purple);
  }

  p {
    opacity: 0.7;
    font-size: 0.8rem;
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
