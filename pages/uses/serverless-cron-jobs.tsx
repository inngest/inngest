import styled from "@emotion/styled";
import React, { useState } from "react";
import Button from "src/shared/Button";
import Nav from "src/shared/nav";
import Footer from "src/shared/footer";
import CodeWindow from "src/shared/CodeWindow";

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

const snippet = `
import { createScheduledFunction } from "inngest";

createScheduledFunction(
  "Daily check",
  "0 0 * * *",
  async () => {
    // This function will run every day at Midnight, UTC.
  },
);
`

export default function Template() {
  const [demo, setDemo] = useState(false);

  return (
    <div>
      <Nav sticky={true} nodemo />
      <div className="container mx-auto py-32 flex flex-row">
        <div className="basis-1/2 px-6">
          <h1>Serverless cron jobs, made simple</h1>
          <p className="pt-6 subheading">
            Define and write scheduled functions in your existing projects with a single line of
            code, no infrastructure or configuration required.
          </p>
          <div className="flex flex-row pt-12">
            <Button kind="primary" href="/sign-up?ref=js-hero">
              Get started
            </Button>
            <Button kind="outline" href="https://www.github.com/inngest/inngest-js" target="_blank">
              Get the SDK
            </Button>
          </div>
        </div>
        <div className="basis-1/2 px-6 flex items-center">
            <CodeWindow
              className="transform-iso shadow-xl relative z-10"
              filename={`scheduled/function.ts`}
              snippet={snippet}
            />
        </div>
      </div>

      <div className="container mx-auto">
        <h2 className="text-center">Designed for Developers</h2>
        <p className="text-center pt-2 pb-24">
          Inngest is the easiest way to build scheduled jobs in your app, no matter what framework or platform you use.
        </p>

        <div className="grid grid-cols-3 text-center pb-32 gap-6">
          <div className="bg-white rounded shadow-xl p-8">
            <h3 className="mb-2">Easy to use</h3>
            <p>Create scheduled functions and cron jobs with a single line of code</p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <h3 className="mb-2">Declarative</h3>
            <p>Define functions and schedules together in one place for easy maintenance</p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <h3 className="mb-2">Serverless</h3>
            <p>Scheduled functions run without servers or configuration - no setup required</p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <h3 className="mb-2">Great DX</h3>
            <p>Local development-only UI to inspect functions and their schedules</p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <h3 className="mb-2">Any platform</h3>
            <p>Keep your code together by deploying to your platform choice</p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <h3 className="mb-2">Reliable</h3>
            <p>If your job fails we'll rerun it multiple times without any work your side</p>
          </div>
        </div>
      </div>

      <div className="container mx-auto">
        <h2 className="text-center">How it works</h2>
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

