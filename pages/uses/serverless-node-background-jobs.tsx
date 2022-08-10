import styled from "@emotion/styled";
import React from "react";
import Button from "src/shared/Button";
import Nav from "src/shared/nav";

export default function Template() {
  return (
    <div>
      <Nav sticky={true} nodemo />
      <div className="container mx-auto py-32 flex flex-row">
        <div className="basis-1/2 px-6">
          <h1>Serverless background jobs for Node & Deno</h1>
          <p className="pt-6">
            Build, test, then deploy background jobs and scheduled tasks without
            worrying about infrastructure or queues — so you can focus on your
            product.
          </p>
          <div className="flex flex-row py-12">
            <Button kind="primary">Sign up</Button>
            <Button kind="outline">Read the docs</Button>
          </div>
        </div>
        <div className="basis-1/2 px-6">img</div>
      </div>

      <div className="container mx-auto">
        <h2 className="text-center">Designed for Developers</h2>
        <p className="text-center pt-2 pb-24">
          Develop, test, and deploy background tasks for Node and Deno using a
          single CLI built for developer productivity.
        </p>

        <Developers className="grid grid-cols-2 gap-8">
          <div>
            <img src="https://via.placeholder.com/900x450" />
          </div>
          <div className="flex flex-col justify-center">
            <h3 className="pb-2">
              Simple development: <code>inngest init</code>
            </h3>
            <p>
              Easily write background jobs and scheduled tasks using Node, Deno,
              Typesript, Reason, Elm — or any other language in your stack. A
              single command scaffolds the entire serverless function ready to
              test.
            </p>
          </div>

          <div>
            <img src="https://via.placeholder.com/900x450" />
          </div>
          <div className="flex flex-col justify-center">
            <h3 className="pb-2">
              Local testing: <code>inngest run</code>
            </h3>
            <p>
              Test your functions locally with a single command, using randomly
              generated data or real production data via replay.
            </p>
            <ul className="space-y-1">
              <li>Local testing without any setup</li>
              <li>Test with real production data</li>
            </ul>
          </div>

          <div>
            <img src="https://via.placeholder.com/900x450" />
          </div>
          <div className="flex flex-col justify-center">
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
              <li>CI/CD built in</li>
            </ul>
          </div>
        </Developers>
      </div>

      <div className="container mx-auto flex py-32 justify-center">
        <div className="flex flex-col justify-center align-center text-center pr-24">
          <p>“Sooooo much easier than AWS”</p>
          <a href="https://btwn.com" rel="nofollow" target="_blank">
            Between
          </a>
        </div>
        <div>
          <Button kind="primary">Start building today</Button>
        </div>
      </div>

      <div className="container mx-auto pb-24">
        <div className="text-center mx-auto max-w-2xl pb-24">
          <h2 className="pb-2">Not your ordinary task scheduler</h2>
          <p>
              Inngest’s platform provides cloud-native, serverless features
              essential for modern development, allowing you to build complex products without servers,
            configuration, or complexity.
          </p>
        </div>

        <FeatureGrid className="grid grid-cols-3 gap-8 gap-y-16 pb-32">
          <div>
            <h3>Scalable</h3>
            <p>Functions run and scale automatically based off of incoming events and webhooks, without specifying or managing queues</p>
          </div>
          <div>
            <h3>Easy to use</h3>
            <p>Build and locally test functions without any extra work, with single commands to invoke and deploy functions</p>
          </div>
          <div>
            <h3>Fully versioned</h3>
            <p>Every function is fully versioned, with test and production environments provided for each account</p>
          </div>
          <div>
            <h3>Background tasks & webhooks</h3>
            <p>Run any logic in the background via a single JSON event — without worrying about servers or private APIs</p>
          </div>
          <div>
            <h3>Scheduled functions</h3>
            <p>Build and test serverless functions which run on a schedule, without managing infra or crons</p>
          </div>
          <div>
            <h3>User attribution</h3>
            <p>Attribute each function directly to the relevant user — whether it's an internal employee or a customer</p>
          </div>
        </FeatureGrid>

        <div className="flex">
          <div className="basis-1/2 flex flex-col justify-center pr-8">
            <h2 className="pb-6">Fully serverlesss, locally testable, made&nbsp;for&nbsp;collaboration.</h2>
            <p>People use Inngest to reliably run background work, serverless functions, and scheduled jobs across for a variety of use cases — including building out internal tasks for the wider team.</p>
            <p className="mt-2 pb-10">Common examples include webhook management, background jobs, scheduled tasks, and end-to-end automation.</p>
            <div><Button kind="primary">Start building today</Button></div>
          </div>
          <div className="pl-8">
            <img src="https://via.placeholder.com/900x550" />
          </div>
        </div>
      </div>
    </div>
  );
}

const Developers = styled.div`
  h3 code {
    color: var(--color-iris-60);
    margin-left: 0.25rem;
  }
`;

const FeatureGrid = styled.div`
  text-align: center;

  > div {
  }

  h3 {
    font-weight: normal;
    padding-bottom: 0.25rem;
    font-size: 1.25rem;
  }
`
