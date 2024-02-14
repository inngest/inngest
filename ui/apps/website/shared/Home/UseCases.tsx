import { useState } from 'react';
import Link from 'next/link';
import clsx from 'clsx';

import CodeWindow from '../CodeWindow';
import Container from '../layout/Container';
import Heading from './Heading';

const snippetDurableWorkflow = `
export const processVideo = inngest.createFunction(
  fnOptions, fnListener,
  async ({ event, step }) => {
    const transcript = await step.run('transcribe-video',
      async () => deepgram.transcribe(event.data.videoUrl)
    )
    const summary = await step.run('summarize-transcript',
      async () => llm.createCompletion({
        model: "gpt-3.5-turbo",
        prompt: createSummaryPrompt(transcript),
      })
    )
    await step.run('write-to-db',
      async () => db.videoSummaries.upsert({
        videoId: event.data.videoId,
        transcript,
        summary,
      })
    )
  }
);
`;

const snippetAI = `
export const userWorkflow = inngest.createFunction(
  fnOptions, fnListener,
  async ({ event, step }) => {
    const similar = await step.run("query-vectordb",
      async () => {
        const embedding = createEmedding(event.data.input);
        return await index.query({
          vector: embedding, topK: 3
        }).matches;
      });
    const data = await step.run("generate-llm-response",
      async () =>
        await llm.createCompletion({
          model: "gpt-3.5-turbo",
          prompt: createAgentPrompt(similar),
        });
      );
    await step.run("save-to-db", async () => {
      await db.summaries.create({
        requestID: event.data.requestID, data
      });
    });
  }
);
`;

const snippetBackgroundJobs = `
export const welcomeEmail = inngest.createFunction(
  {
    name: "Send welcome email",
    id: "send-welcome-email",
    concurrency: {
      limit: 10,
    }
  },
  {
    event: "clerk/user.created"
  },
  async ({ event, step }) => {
    await step.run('send-email', async () => {
      return await resend.emails.send({
        from: 'noreply@inngest.com',
        to: event.user.email,
        subject: "Welcome to Inngest!",
        react: WelcomeEmail(),
      });
    });
  }
);
`;

const snippetWorkflowEngine = `
export const engine = inngest.createFunction(
  fnOptions, fnListener,
  async ({ event, step }) => {
    const workflow = await step.run('load-workflow',
      async () =>
        db.workflows.find({
          where: { id: event.data.workflowID }
        });
    );

    for (let action of workflow) {
      const result = await step.run("run-action",
        async () => {
          return runAction(event, action);
        }
      );
    }
  }
);
`;

const content = [
  {
    title: 'Durable workflows',
    content: (
      <p>
        Combine any complex series of tasks into a single reliable workflow. Each task becomes a
        “step” which is automatically retried when errors happen.
      </p>
    ),
    bullets: [
      <>Run steps in parallel or series or call a child workflow.</>,
      <>Add durable sleep to pause your workflow for days or weeks at a time.</>,
      <>Visually debug the entire workflow without having to parse logs and connect the dots.</>,
    ],
    snippet: snippetDurableWorkflow,
    href: '/uses/durable-workflows?ref=homepage',
  },
  {
    title: 'AI + LLMs',
    content: <p>Chain calls to LLMs or any AI API reliably without worrying about glue code.</p>,
    bullets: [
      <>Handle complex text-generation with chain-based post-processing.</>,
      <>Wrap steps to run exactly once to reduce extra, expensive API calls.</>,
      <>Load data from databases and vector stores without complex interfaces or adapters.</>,
    ],
    snippet: snippetAI,
    href: '/ai?ref=homepage',
  },
  {
    title: 'Background jobs',
    content: <p>Write declarative background jobs without queues or infrastructure.</p>,
    bullets: [
      <>Run your code in serverless, servers, or on the edge.</>,
      <>Fan-out work to multiple functions from a single event trigger.</>,
      <>Logs and observability metrics out-of-the-box.</>,
    ],
    snippet: snippetBackgroundJobs,
    href: '/uses/serverless-node-background-jobs?ref=homepage',
  },
  {
    title: 'Workflow engines',
    content: (
      <p>
        Create a user-customizable workflow engine right in your product without having to build the
        engine itself.
      </p>
    ),
    bullets: [
      <>Build linear or complex DAG-workflows with our SDK's primitives.</>,
      <>Handle concurrency, prioritization and debounce in line with your user's limits.</>,
      <>Easily audit, observe and scale your product without breaking a sweat.</>,
    ],
    snippet: snippetWorkflowEngine,
    href: '/uses/workflow-engine?ref=homepage',
  },
];

export default function UseCases() {
  return (
    <Container className="mt-12">
      <Heading
        title="Ship the impossible. Today."
        lede={
          <>Using simple primitives you can build the most complex systems without the stress.</>
        }
        className="text-center"
      />

      <UseCaseGrid />

      {/* TODO - List all other use cases with links */}
    </Container>
  );
}

function UseCaseGrid() {
  const [selected, setSelected] = useState(0);
  const selectedContent = content[selected];
  return (
    <div>
      <div className="mx-auto my-12 flex flex-wrap justify-center gap-x-4 gap-y-2">
        {content.map(({ title, content, bullets, href }, idx) => (
          <Option key={idx} isSelected={selected === idx} onClick={() => setSelected(idx)}>
            {title}
          </Option>
        ))}
      </div>
      <div className="grid-rows-auto mx-auto mt-12 grid max-w-5xl grid-cols-1 gap-px rounded-md bg-gradient-to-tl from-green-800/60 via-orange-300/60 to-rose-900/60 p-px shadow-[0_10px_100px_0_rgba(52,211,153,0.10)] md:grid-cols-8 lg:grid-cols-7">
        <div className="flex flex-col gap-px md:col-span-4 md:rounded-tl-md">
          <div className="bg-slate-1000 grow p-8 md:rounded-tl-md">{selectedContent.content}</div>
          <div className="bg-slate-1000 grow p-8">
            <ul className="ml-4 flex grow list-disc flex-col gap-2">
              {selectedContent.bullets.map((b) => (
                <li>{b}</li>
              ))}
            </ul>
          </div>
          <div className="bg-slate-1000 p-8 md:rounded-bl-md">
            <Link
              href={`${selectedContent.href}?ref=homepage-use-cases}`}
              className="mx-auto whitespace-nowrap rounded-md border border-slate-800 bg-slate-800 px-6 py-2 font-medium text-white transition-all hover:border-slate-600 hover:bg-slate-500/10 hover:bg-slate-600"
            >
              <span className="hidden md:inline">
                Learn about {selectedContent.title} with Inngest
              </span>
              <span className="md:hidden">Learn more</span> →
            </Link>
          </div>
        </div>

        {/* The min height here is for the longest code snippet that we show */}
        <div className="bg-slate-1000 md:col-span-4 md:min-h-[492px] md:rounded-r-md lg:col-span-3">
          <CodeWindow snippet={selectedContent.snippet} className="border-0 bg-transparent" />
        </div>
      </div>
    </div>
  );
}

function Option({ isSelected = false, onClick, children }) {
  return (
    <button
      onClick={onClick}
      className={clsx(
        `whitespace-nowrap rounded-full border border border-slate-800 px-4 py-2 text-base font-semibold text-white hover:border-slate-600 hover:bg-slate-500/10 sm:text-lg`,
        isSelected && 'bg-slate-100 text-slate-900 hover:bg-slate-100/90'
      )}
    >
      {children}
    </button>
  );
}
