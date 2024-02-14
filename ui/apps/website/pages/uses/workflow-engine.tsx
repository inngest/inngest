import { useEffect } from 'react';
import { GetStaticProps } from 'next';
import Link from 'next/link';
import { ChevronRightIcon } from '@heroicons/react/20/solid';
import { useRive } from '@rive-app/react-canvas';
import CodeWindow from 'src/shared/CodeWindow';
import Footer from 'src/shared/Footer';
import Header from 'src/shared/Header';
import Quote from 'src/shared/Home/Quote';
import Check from 'src/shared/Icons/Check';
import Container from 'src/shared/layout/Container';
import PageContainer from 'src/shared/layout/PageContainer';

export const getStaticProps: GetStaticProps = async (ctx) => {
  return {
    props: {
      designVersion: '2',
    },
  };
};

export default function workflowEngine() {
  const { rive, RiveComponent } = useRive({
    src: '/assets/animations/workflows.riv',
    stateMachines: 'state',
    autoplay: false,
  });

  useEffect(() => {
    rive && window.setTimeout(() => rive && rive.play(), 500);
  }, [rive]);

  return (
    <PageContainer>
      <Header />

      <Container>
        <div className="justify-between gap-2 py-24 lg:items-center lg:py-48">
          <div className="grid grid-cols-1 content-center lg:grid-cols-2 lg:gap-40">
            <div>
              <h1
                className="
                mb-8 text-4xl font-semibold
                leading-[48px] tracking-[-2px]
                text-slate-50 sm:text-5xl
                sm:leading-[58px] lg:text-6xl lg:leading-[68px]
              "
              >
                Launch customizable workflows, in&nbsp;weeks
              </h1>

              <p className="text-lg leading-8 text-slate-200">
                Build powerful customizable workflows directly in your product using Inngest as the
                reliable orchestration engine. Develop locally and ship to your existing production
                systems ready for any scale.
              </p>

              <ul className="my-8 text-lg font-medium leading-8">
                <li className="flex items-center">
                  <Check size={20} className="mr-2" /> Integrate directly into your existing code
                </li>
                <li className="flex items-center">
                  <Check size={20} className="mr-2" /> Powerful, customizable, and observable
                </li>
                <li className="flex items-center">
                  <Check size={20} className="mr-2" /> Operate at scale
                </li>
              </ul>
            </div>
            <div className="flex items-center justify-center">
              <div className="h-[405px] w-full lg:w-[500px]">
                <RiveComponent />
              </div>
            </div>
          </div>
        </div>
      </Container>

      <Container>
        <div className="grid lg:grid-cols-2 lg:gap-40">
          <div className="flex hidden items-center justify-center lg:block">
            <img src="/assets/florianworks.jpg" className="rounded-lg" />
          </div>

          <div>
            <span className="text-2xs uppercase tracking-[.25em]">Case study</span>
            <h2 className="my-4 text-3xl font-semibold">
              Florian Works: zero to building a mission-critical workflow engine for fire
              departments
            </h2>
            <p className="mb-3">
              Florian Works develops custom-built software products for fire departments,
              incorporating custom workflows built directly on top of Inngest to ship reliable
              products faster and easier than ever before.
            </p>
            <p>
              Utilizing Inngest's core workflow engine and primitives such as{' '}
              <code className="text-sm">step.waitForEvent</code>, FlorianWorks ships scheduling,
              roster management, a rules engine, and finance management without spending effort
              developing custom distributed systems primitives or reliability concerns.
            </p>
            <ul className="my-3 leading-8">
              <li className="flex items-center">
                <Check size={14} className="mr-2" /> Development on core business logic only
              </li>
              <li className="flex items-center">
                <Check size={14} className="mr-2" /> Auditable, logged, secure workflows
              </li>
              <li className="flex items-center">
                <Check size={14} className="mr-2" /> Zero additional infrastructure required
              </li>
            </ul>
            <div className="mt-8">
              <Link
                href="/customers/florian-works"
                className="mx-auto whitespace-nowrap rounded-md border border-slate-800 bg-slate-800 px-6 py-2 font-medium text-white transition-all hover:border-slate-600 hover:bg-slate-500/10 hover:bg-slate-600"
              >
                Read customer case study
              </Link>
            </div>
          </div>
        </div>
      </Container>

      <Container>
        <Quote
          text="Inngest is a great platform to build reliability into your long
        running tasks without drowning in complexity."
          attribution={{
            name: 'Ozan Şener',
            title: 'Principal Engineer',
            avatar: '/assets/quotes/osenergy.jpeg',
          }}
          className="my-36"
        />
      </Container>

      <Container className="mb-24 mt-24">
        <div className="my-14 grid gap-40 lg:grid-cols-2">
          <div>
            <h2 className="my-4 text-3xl font-semibold">Fully customizable, durable workflows</h2>
            <p className="my-4">
              You bring the application code, we bring the engine. Allow your own users to create
              workflows composed of reusable logic that you define. Our engine runs workflows as
              steps, taking care of scale, orchestration, idempotency, retries, and observability
              for you.
            </p>
            <p>
              Build simple linear workflows or complex DAG-based workflows with parallelism and
              fan-in out of the box. Leverage our step primitives for human-in-the-loop or paused
              functions which automatically resume based off of conditions being met.
            </p>

            <div className="mt-8 flex">
              <Check size={14} className="mr-2 mt-1 inline" />
              <div className="flex-1">
                <strong className="font-semibold">Concurrency, rate limiting and debounce</strong>
                &nbsp;controls built in, with custom keys or controlling your own user's&nbsp;limits
              </div>
            </div>
            <div className="mt-4 flex">
              <Check size={14} className="mr-2 mt-1 inline" />
              <div className="flex-1">
                <strong className="font-semibold">Reliably run any code</strong>
                &nbsp;in any step, with retries and error handling automatically&nbsp;managed
              </div>
            </div>
            <div className="mt-4 flex">
              <Check size={14} className="mr-2 mt-1 inline" />
              <div className="flex-1">
                <strong className="font-semibold">Auditable, observable, and scalable</strong>
                &nbsp;handling tens of thousands of requests per second with real time metrics
              </div>
            </div>

            <div className="flex w-full flex-col items-center justify-center gap-8 pt-12 lg:flex-row lg:py-28">
              <div>
                <Link
                  href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=workflows`}
                  className="flex flex-row items-center whitespace-nowrap rounded-md bg-indigo-500 px-11 py-3.5 font-medium text-white transition-all hover:bg-indigo-400"
                >
                  Get started
                  <ChevronRightIcon className="relative top-px h-5 transition-transform duration-150 group-hover:translate-x-1" />
                </Link>
              </div>
              <Link
                href="/contact"
                className="group flex items-center gap-1 whitespace-nowrap rounded-md border border-transparent bg-transparent px-11 py-3.5 text-indigo-200 transition-all hover:border-slate-800"
              >
                Contact us
              </Link>
            </div>
          </div>

          <div>
            <CodeWindow
              header={
                <div className="flex px-5 py-2">
                  <div className="py-1 text-sm font-light text-slate-400">workflow.ts</div>
                </div>
              }
              snippet={stackWorkflows}
              showLineNumbers={true}
            />
          </div>
        </div>
      </Container>
      <Footer ctaRef={`use-case-workflow`} />
    </PageContainer>
  );
}

const stackWorkflows = `
import { runAction } from "@/actions";
import { inngest } from "@/inngest";

const fnOptions = {
  id: "user-workflows",
  // limit to 10 workflows for each tenant in your system.
  concurrency: {
    limit: 10,
    key: "event.data.account_id",
  },
};

const fnListener = { event: "api/workflow.invoked" };

// Create a durable function which runs user defined workflows any time
// the "api/workflow.invoked" event is received.  This loads the specified
// user's workflow from your own system, and executes each step of the flow.
export const userWorkflow = inngest.createFunction(
  fnOptions,
  fnListener,
  async ({ event, step }) => {
    const workflow = await step.run("load-workflow", async () => {
      return db.workflows.find({ where: { id: event.data.workflowID } });
    });

    // Iterate over a simple stack, or create a graph and iteerate over a full
    // blown DAG whioch a user can define.
    for (let action of workflow) {
      const result = await step.run("run-action", async () => {
        return runAction(event, action);
      });
    }
  }
);
`;
