import styled from "@emotion/styled";
import React, { useState } from "react";
import Button from "src/shared/legacy/Button";
import Nav from "src/shared/legacy/nav";
import CodeWindow from "src/shared/legacy/CodeWindow";
import Footer from "src/shared/legacy/Footer";
import Play from "src/shared/Icons/Play";
import IconListStories from "src/stories/IconList.stories";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Serverless background jobs for Node & Deno",
        description:
          "Build, test, then deploy background tasks without worrying about infrastructure or queues — so you can focus on your product.",
      },
    },
  };
}

const snippet = `
import { createScheduledFunction } from "inngest";

createFunction(
  "After signup",
  "auth/user.created",
  async ({ event }) => {
    // This function runs in the background every time
    // the "auth/user.created" event is received.
  },
);
`;

const fullSnippet = `
import { createScheduledFunction } from "inngest";

createFunction(
  "After signup",
  "auth/user.created",
  async ({ event }) => {
    // Instead of your signup API triggering activation
    // emails and setting up user accounts, your API
    // offloads this work into the background by
    // triggering an event which calls this function.
    await sendActivationEmail({
      email: event.user.email,
      name: event.user.name
    });
    await createChurnCampaign({ id: event.user.id });
  },
);
`;

const sendSnippet = `
import { Inngest } from 'inngest';

// Create a new client for sending events.
export const client = new Inngest({ name: "Your app name" });

// Send an event to Inngest, which triggers any background
// function that listens to this event.
client.send({
  name: "auth/user.created",
  data: {
    plan: "The super awesome new plan",
  },
  user: {
    id: "8f2bc",
    email: "user@example.com",
    name: "Super Awesome User",
  }
});
`;

export default function Template() {
  const [demo, setDemo] = useState(false);

  return (
    <div>
      <Nav sticky={true} nodemo />
      <div className="container mx-auto py-32 flex flex-row">
        <div className="basis-1/2 px-6">
          <h1>Background tasks, without the queues or workers</h1>
          <p className="pt-6 subheading">
            Build serverless background tasks without any queues,
            infrastructure, or config.
          </p>
          <p className="mt-2">
            Inngest allows you to easily offload work from your APIs to increase
            your app's speed &mdash; without changing your project or building
            complex architectures.
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
        <div className="basis-1/2 px-6 flex items-center">
          <CodeWindow
            className="transform-iso shadow-xl relative z-10"
            filename={`functions/after_signup.ts`}
            snippet={snippet}
          />
        </div>
      </div>

      <div className="container mx-auto">
        <h2 className="text-center">Running in seconds</h2>

        <p className="text-center pt-2 pb-24">
          Inngest's SDK allows you to write and deploy background jobs with a
          single line of code. Here's how it works:
        </p>

        <Developers className="grid grid-cols-2 gap-8 gap-y-16">
          <div className="flex flex-col justify-center">
            <CodeWindow
              className="shadow-xl border relative z-10"
              filename={`functions/after_signup.ts`}
              snippet={fullSnippet}
            />
          </div>
          <div className="flex flex-col justify-center">
            <h3 className="pb-2">1. Define your jobs</h3>
            <p>
              Write your background jobs using regular TypeScript or JavaScript,
              then use our SDK to define which events trigger the job.
            </p>
            <p className="mt-2">
              It takes a single line of code to specify how an event runs in the
              background, and a single line of code to provide an API that
              serves all background functions together.
            </p>
          </div>

          <div className="flex flex-col justify-center">
            <CodeWindow
              className="shadow-xl border relative z-10"
              filename={`api/signup.ts`}
              snippet={sendSnippet}
            />
          </div>
          <div className="flex flex-col justify-center">
            <h3 className="pb-2">2. Run jobs via events</h3>
            <p>
              Sending events to Inngest automatically triggers background jobs
              which subscribe to that event — without any queues, databases, or
              configuration.
            </p>
            <p className="mt-2">
              This allows you to create <b>single background jobs</b> or{" "}
              <b>fan-out jobs that run in parallel</b>, triggered via a single
              event.
            </p>
            <p className="mt-2">
              Events are fully typed, so you can guarantee that the data you
              send and receive is correct.
            </p>
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
