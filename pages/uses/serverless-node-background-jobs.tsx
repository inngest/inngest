import styled from "@emotion/styled";
import React, { useState } from "react";
import Button from "src/shared/Button";
import Nav from "src/shared/nav";
import Footer from "src/shared/footer";
import Play from "src/shared/Icons/Play";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Serverless background jobs for Node & Deno",
        description:
          "Build, test, then deploy background jobs and scheduled tasks without worrying about infrastructure or queues — so you can focus on your product.",
      },
    },
  };
}

export default function Template() {
  const [demo, setDemo] = useState(false);

  return (
    <div>
      <Nav sticky={true} nodemo />
      <div className="container mx-auto py-32 flex flex-row">
        <div className="basis-1/2 px-6">
          <h1>Serverless background jobs for Node & Deno</h1>
          <p className="pt-6 subheading">
            Build, test, then deploy background jobs and scheduled tasks without
            worrying about infrastructure or queues — so you can focus on your
            product.
          </p>
          <div className="flex flex-row pt-12">
            <Button kind="primary" href="/sign-up?ref=js-hero">
              Sign up
            </Button>
            <Button kind="outline" href="/docs?ref=js-hero">
              Read the docs
            </Button>
          </div>
        </div>
        <VidPlaceholder className="basis-1/2 px-6 flex items-center">
          <button
            className="flex items-center justify-center"
            onClick={() => setDemo(true)}
          >
            <Play outline={false} fill="var(--primary-color)" size={80} />
          </button>
          <img src="/assets/homepage/cli-3-commands.png" />
        </VidPlaceholder>
      </div>

      <div className="container mx-auto">
        <h2 className="text-center">Designed for Developers</h2>
        <p className="text-center pt-2 pb-24">
          Develop, test, and deploy background tasks for Node and Deno using a
          single CLI built for developer productivity.
        </p>

        <Developers className="grid grid-cols-2 gap-8 gap-y-16">
          <div className="flex flex-col justify-center">
            <img src="/assets/product/cli-init.png" />
          </div>
          <div className="flex flex-col justify-center">
            <small>Designed for ease of use</small>
            <h3 className="pb-2">
              Simple development: <code>inngest init</code>
            </h3>
            <p>
              Easily write background jobs and scheduled tasks using Node, Deno,
              Typesript, Reason, Elm — or any other language in your stack. A
              single command scaffolds the entire serverless function ready to
              test.
            </p>
            <ul className="space-y-1">
              <li>Create scheduled or background jobs</li>
              <li>Easily build complex step functions using any languages</li>
            </ul>
          </div>

          <div className="flex flex-col justify-center">
            <img src="/assets/product/cli-run.png" />
          </div>
          <div className="flex flex-col justify-center">
            <small>Designed for local development</small>
            <h3 className="pb-2">
              Local testing: <code>inngest run</code>
            </h3>
            <p>
              Test your functions locally with a single command, using randomly
              generated data or real production data via replay, then run a real
              job queue in your project with zero infra via{" "}
              <code>inngest dev</code>.
            </p>
            <ul className="space-y-1">
              <li>Locally run without any setup</li>
              <li>Test with real production data</li>
              <li>
                Local UI with step function debugger <small>Coming soon</small>
              </li>
            </ul>
          </div>

          <div className="flex flex-col justify-center">
            <img src="/assets/product/cli-deploy.png" />
          </div>

          <div className="flex flex-col justify-center">
            <small>Designed to scale</small>
            <h3 className="pb-2">
              One-command deploy: <code>inngest deploy</code>
            </h3>
            <p>
              Roll out new background jobs and scheduled tasks using a single
              command — without setting up a single server, queue, or Redis
              instance, and without changing your app.
            </p>
            <ul className="space-y-1">
              <li>CI/CD built in</li>
              <li>Immediate rollbacks</li>
              <li>Deploy functions as internal tools</li>
            </ul>
          </div>
        </Developers>
      </div>

      <div className="container mx-auto flex py-32 justify-center">
        <div className="flex flex-col justify-center align-center text-center pr-24">
          <p>
            <i>“Sooooo much easier than AWS”</i>
          </p>
          <small>Between</small>
        </div>
        <div>
          <Button kind="primary" href="/sign-up?ref=collab">
            Start building today
          </Button>
        </div>
      </div>

      <Features>
        <div className="container mx-auto py-24">
          <div className="text-center mx-auto max-w-2xl pb-24">
            <h2 className="pb-2">Not your ordinary task scheduler</h2>
            <p>
              Inngest’s platform provides cloud-native, serverless features
              essential for modern development, allowing you to build complex
              products without servers, configuration, or complexity.
            </p>
          </div>

          <FeatureGrid className="grid grid-cols-3 gap-8 gap-y-16 pb-32">
            <div>
              <h3>Scalable</h3>
              <p>
                Functions run and scale automatically based off of incoming
                events and webhooks, without specifying or managing queues
              </p>
            </div>
            <div>
              <h3>Easy to use</h3>
              <p>
                Build and locally test functions without any extra work, with
                single commands to invoke and deploy functions
              </p>
            </div>
            <div>
              <h3>Fully versioned</h3>
              <p>
                Every function is fully versioned, with test and production
                environments provided for each account
              </p>
            </div>
            <div>
              <h3>Background tasks & webhooks</h3>
              <p>
                Run any logic in the background via a single JSON event —
                without worrying about servers or private APIs
              </p>
            </div>
            <div>
              <h3>Scheduled functions</h3>
              <p>
                Build and test serverless functions which run on a schedule,
                without managing infra or crons
              </p>
            </div>
            <div>
              <h3>User attribution</h3>
              <p>
                Attribute each function directly to the relevant user — whether
                it's an internal employee or a customer
              </p>
            </div>
          </FeatureGrid>

          <div className="flex">
            <div className="basis-1/2 flex flex-col justify-center pr-8">
              <h2 className="pb-6">
                Fully serverlesss, locally testable, made
                for&nbsp;collaboration.
              </h2>
              <p>
                People use Inngest to reliably run background work, serverless
                functions, and scheduled jobs across for a variety of use cases
                — including building out internal tasks for the wider team.
              </p>
              <p className="mt-2 pb-10">
                Common examples include webhook management, background jobs,
                scheduled tasks, and end-to-end automation.
              </p>
              <div className="flex">
                <Button kind="primary" href="/sign-up?ref=js-mid">
                  Start building today
                </Button>
              </div>
            </div>
            <div className="basis-1/2 pl-8">
              <img src="/assets/homepage/admin-ui-screenshot.png" />
            </div>
          </div>
        </div>
      </Features>

      <div className="container mx-auto text-center pt-32">
        <h2 className="max-w-lg mx-auto pb-6">
          Invoke many background functions with a single HTTP POST
        </h2>
        <p className="max-w-2xl mx-auto pb-12">
          Inngest’s core difference is that it’s <i>event-driven</i>. Send a
          single JSON event to Inngest and run any number of functions
          automatically, and we’ll statically type-check the JSON payload then
          store each event for logging and backtesting. It's way better than
          old-school RPC.
        </p>

        {/*
        <div
          className="aspect-video max-w-4xl mx-auto"
          style={{ background: "#ccc" }}
        />
        */}

        <div className="flex justify-center pt-12">
          <Button kind="primary" href="/sign-up?ref=js-footer">
            Sign up
          </Button>
          <Button kind="outline" href="/docs?ref=js-footer">
            Read the docs
          </Button>
        </div>
      </div>

      <Quote className="container mx-auto max-w-xl pt-24 text-center space-y-3">
        <q>
          This is 100% the dev/prod parity that we’re lacking for queue-based
          systems.
        </q>
        <p>Staff Engineer at Buffer</p>
      </Quote>

      <Footer />

      {demo && (
        <Demo
          className="flex justify-center items-center"
          onClick={() => {
            setDemo(false);
          }}
        >
          <div className="container aspect-video mx-auto max-w-2xl flex">
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

const VidPlaceholder = styled.div`
  position: relative;

  button {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;

    svg {
      box-shadow: 0 0 40px var(--primary-color);
      border-radius: 60px;
      transition: all 0.3s;
    }

    &:hover svg {
      box-shadow: 0 0 80px 20px var(--primary-color);
    }
  }
`;

const Developers = styled.div`
  h3 code {
    color: var(--color-iris-60);
    margin-left: 0.25rem;
  }

  div > small {
    font-weight: bold;
    opacity: 0.4;
  }
`;

const Features = styled.div`
  position: relative;
  &:before,
  &:after {
    content: "";
    display: block;
    height: 100%;
    width: 100%;
    position: absolute;
    top: 0;
    left: 0;
    z-index: 0;
  }
  &:before {
    background: var(--bg-feint-color) url(/assets/grid-bg.png) repeat 0 0;
    opacity: 0.8;
  }
  &:after {
    background: radial-gradient(
      50% 50% at 50% 50%,
      rgba(247, 248, 249, 0) 0%,
      #f7f8f9 100%
    );
  }
  > div {
    position: relative;
    z-index: 1;
  }

  .subheading {
    color: #556987;
  }
`;

const FeatureGrid = styled.div`
  text-align: center;

  h3 {
    font-weight: normal;
    padding-bottom: 0.25rem;
    font-size: 1.25rem;
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
