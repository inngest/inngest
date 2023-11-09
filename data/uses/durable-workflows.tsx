import type { UseCase } from "../../pages/uses/[case]";

export const data: UseCase = {
  title: "Durable Workflows",
  lede: `Write complex workflows as code and let Inngest handle the rest. Inngest manages state, retries, logging and observability for you.`,
  heroImage: "/assets/use-cases/function-timeline.png",
  keyFeatures: [
    {
      title: "Run steps in series or parallel",
      img: "series-v-parallel.png",
      description:
        "Easily run parts of your workflow code in parallel, event on serverless. Always retried automatically on error.",
    },
    {
      title: "Durable sleep for hours *or* weeks",
      img: "durable-sleep.png",
      description:
        "Pause your function and schedule to resume it after a specific period of time. Your code stops running and Inngest resumes it when the time is right.",
    },
    {
      title: "Visual debugging and observability",
      img: "debug-steps.png",
      description:
        "Our visual function timeline UI makes debugging easier than ever. See exactly what happened in your function, and when without grepping logs.",
    },
  ],
  codeSection: {
    title: "Create complex workflows with simple primitives",
    examples: [
      {
        title: "Chain steps that automatically retry",
        steps: [
          "Decouple logic into discrete steps",
          "Steps that fail are retried automatically",
          "Steps that succeed are never re-run, saving time and money",
          "Step output is captured as state, and can be used in subsequent steps",
        ],
        description:
          "You can skip managing multiple queues and workers and persisting state of jobs in your database. Inngest takes care of that for you.",
        code: `
          export const processVideo = inngest.createFunction(
            {
              name: "Process video upload", id: "process-video",
            },
            { event: "video.uploaded" },
            async ({ event, step }) => {
              const transcript = await step.run('transcribe-video', async () => {
                return deepgram.transcribe(event.data.videoUrl);
              });
              const summary = await step.run('summarize-transcript', async () => {
                return llm.createCompletion({
                  model: "gpt-3.5-turbo",
                  prompt: createSummaryPrompt(transcript),
                });
              });
              await step.run('write-to-db', async () => {
                await db.videoSummaries.upsert({
                  videoId: event.data.videoId,
                  transcript,
                  summary,
                });
              });
            }
          )`,
      },
      {
        title: "Long running, durable workflows",
        steps: [
          "Trigger workflows with events",
          "Sleep for hours, days or longer",
          "Your code stops running and Inngest resumes it when the time is right",
        ],
        description:
          "Write idomatic code that might need to execute over a long period of time. There is no need to combine and manage crons, queues, and database state.",
        code: `
          export const handlePayments = inngest.createFunction(
            {
              name: "Handle payments", id: "handle-payments"
            },
            { event: "api/invoice.created" },
            async ({ event, step }) => {
              // Wait until the next billing date
              await step.sleepUntil("wait-for-billing-date", event.data.invoiceDate);

              // Steps automatically retry on error, and only run
              // once on success - automatically, with no work.
              const charge = await step.run("charge", async () => {
                return await stripe.charges.create({
                  amount: event.data.amount,
                });
              });

              await step.run("update-db", async () => {
                await db.payments.upsert(charge);
              });

              await step.run("send-receipt", async () => {
                await resend.emails.send({
                  to: event.user.email,
                  subject: "Your receipt for Inngest",
                });
              });
            }
          );`,
      },
    ],
  },
  featureOverflow: [
    {
      title: "Automatic retries",
      description:
        "Every step of your function is retried whenever it throws an error. Customize the number of retries to ensure your functions are reliably executed.",
      icon: "Retry",
    },
    {
      title: "Durable sleep",
      description:
        "Pause your function for hours, days or weeks with step.sleep() and step.sleepUntil(). Inngest stores the state of your functions and resumes execution automatically exactly when it should.",
      icon: "Scheduled",
    },
    {
      title: "Declarative job cancellation",
      description:
        "Cancel jobs just by sending an event. No need to keep track of running jobs, Inngest can automatically match long running functions with cancellation events to kill jobs declaratively.",
      icon: "Compiling",
    },
    {
      title: "Pause functions for additional input",
      description:
        "Use step.waitForEvent() to pause your function until another event is received. Create human-in the middle workflows or communicate between long running jobs with events.",
      icon: "Steps",
    },
    {
      title: "Replay functions",
      description:
        "Forget dead letter queues. Fix your issues then replay a failed function in a single click.",
      icon: "Power",
    },
  ],
  quote: {
    text: "With Inngest, Vercel and Next.js developers can now go beyond request-response, and orchestrate complex business processes, like a recurring subscription that involves retrying payments, sending notifications, exponential backoff, human-in-the-loop intervention and more.",
    author: "Guillermo Rauch",
    title: "CEO of Vercel",
    avatar: "/assets/customers/vercel-guillermo-rauch.jpg",
  },
  learnMore: {
    description:
      "Dive into our resources and learn how Inngest is the best solution for durable workflows.",
    resources: [
      {
        title: "Quick Start Tutorial",
        description:
          "A step-by-step guide to learn how to build with Inngest in less than 5 minutes.",
        type: "Tutorial",
        href: "/docs/quick-start",
      },
      {
        title: "Running tasks in parallel",
        description:
          "Run code in parallel with automatic retries on serverless or a server.",
        type: "Docs",
        href: "/docs/guides/step-parallelism",
      },
      {
        title:
          "Building an Event Driven Video Processing Workflow with Next.js, tRPC, and Inngest",
        description:
          "How Badass Courses built a self-service video publishing workflow for Kent C. Dodds with AI generated transcripts and subtitles.",
        type: "Blog",
        href: "/blog/nextjs-trpc-inngest",
      },
    ],
  },
};
