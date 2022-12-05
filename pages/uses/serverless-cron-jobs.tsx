import styled from "@emotion/styled";
import React, { useState } from "react";
import Button from "src/shared/legacy/Button";
import Nav from "src/shared/legacy/nav";
import Footer from "src/shared/legacy/Footer";
import CodeWindow from "src/shared/legacy/CodeWindow";
import Developers from "src/shared/Icons/Developers";
import Cube from "src/shared/Icons/Cube";
import Workflow from "src/shared/Icons/Workflow";
import CheckAll from "src/shared/Icons/CheckAll";
import Airplane from "src/shared/Icons/Airplane";
import Retries from "src/shared/Icons/Retries";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Serverless scheduled cron jobs",
        description:
          "Define and write scheduled functions or cron jobs in your existing projects using a single line of code, no infrastructure required.",
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
`;

export default function Template() {
  const [demo, setDemo] = useState(false);

  return (
    <div>
      <Nav sticky={true} nodemo />
      <div className="container mx-auto py-8 lg:py-32 grid lg:grid-cols-2 md:grid-cols-1 gap-12">
        <div className="px-6">
          <h1>Serverless cron jobs, made simple</h1>
          <p className="pt-6 subheading">
            Define and write scheduled functions in your existing projects with
            a single line of code, no infrastructure or configuration required.
          </p>
          <div className="flex flex-row pt-12">
            <Button kind="primary" href="/sign-up?ref=js-hero">
              Get started
            </Button>
            <Button
              kind="outline"
              href="https://www.github.com/inngest/inngest-js"
              target="_blank"
            >
              Get the SDK
            </Button>
          </div>
        </div>
        <div className="px-6 items-center">
          <CodeWindow
            className="transform-iso shadow-xl relative z-10"
            filename={`scheduled/function.ts`}
            snippet={snippet}
          />
        </div>
      </div>

      <div className="container mx-auto py-16">
        <h2 className="text-center">Designed for Developers</h2>
        <p className="text-center pt-2 pb-12 lg:pb-24">
          Inngest is the easiest way to build scheduled jobs in your app, no
          matter what framework or platform you use.
        </p>

        <div className="grid text-center pb-32 gap-6 lg:grid-cols-3 md:grid-cols-1">
          <div className="bg-white rounded shadow-xl p-8">
            <Developers
              fill="var(--color-iris-100"
              className="mx-auto mb-2"
              size={30}
            />
            <h3 className="mb-2">Easy to use</h3>
            <p>
              Create scheduled functions and cron jobs with a single line of
              code
            </p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <Cube
              fill="var(--color-iris-100"
              className="mx-auto mb-2"
              size={30}
            />
            <h3 className="mb-2">Declarative</h3>
            <p>
              Define functions and schedules together in one place for easy
              maintenance
            </p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <Workflow
              fill="var(--color-iris-100"
              className="mx-auto mb-2"
              size={30}
            />
            <h3 className="mb-2">Serverless</h3>
            <p>
              Scheduled functions run without servers or configuration - no
              setup required
            </p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <CheckAll
              fill="var(--color-iris-100"
              className="mx-auto mb-2"
              size={30}
            />
            <h3 className="mb-2">Great DX</h3>
            <p>
              Local development-only UI to inspect functions and their schedules
            </p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <Airplane
              fill="var(--color-iris-100"
              className="mx-auto mb-2"
              size={30}
            />
            <h3 className="mb-2">Any platform</h3>
            <p>Keep your code together by deploying to your platform choice</p>
          </div>
          <div className="bg-white rounded shadow-xl p-8">
            <Retries
              fill="var(--color-iris-100"
              className="mx-auto mb-2"
              size={30}
            />
            <h3 className="mb-2">Reliable</h3>
            <p>
              If your job fails we'll rerun it multiple times without any work
              your side
            </p>
          </div>
        </div>
      </div>

      <div className="container mx-auto">
        <h2 className="text-center mb-16">How it works</h2>
        <div className="grid gap-16 lg:grid-cols-3 md:grid-cols-1">
          <div>
            <h4 className="mb-4">1. Write your functions</h4>
            <p>
              <a href="/docs/functions">Write your scheduled functions</a> using
              regular JS or TS, defined using a single line of code, all served
              via Inngest's handler.
            </p>
          </div>
          <div>
            <h4 className="mb-4">2. Register your URLs</h4>
            <p>
              <a href="/docs/deploy">
                Let Inngest know where your serverless functions are hosted
              </a>{" "}
              — by using our built-in integrations or a single post-deploy API
              call
            </p>
          </div>
          <div>
            <h4 className="mb-4">3. Functions run automatically</h4>
            <p>
              Inngest calls all functions securely and automatically on their
              defined schedule, without any extra setup or servers.
            </p>
          </div>
        </div>
      </div>

      <div className="container mx-auto text-center my-24 w-3/4">
        <p className="text-slate-600 mb-4">
          Build easily using our local SDK UI:
        </p>
        <img
          src="/assets/sdk-ui.png"
          alt="SDK Development UI"
          className="shadow-2xl rounded"
        />
      </div>

      <div className="container mx-auto flex flex-col md:flex-row mt-22 justify-center">
        <div className="flex flex-col justify-center align-center text-center pb-6 md:pb-0 md:pr-24">
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
    </div>
  );
}
