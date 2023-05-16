import type { UseCase } from "../../pages/uses/[case]";

export const data: UseCase = {
  title: "Background tasks, without the queues or workers",
  lede:
    "Build reliable serverless background tasks without setting up any queues or infrastructure.<br/><br/>" +
    "Easily move critical work from your API to a background task in just a few lines of code. Use the Inngest SDK right in your existing codebase.",
  heroImage: "/assets/use-cases/serverless-node-background-jobs/hero-image.png",
  keyFeatures: [
    {
      title: "Use in your existing codebase",
      img: "serverless-queues/left.png",
      description:
        "Define your background jobs in your existing TypeScript or JavaScript codebase and deploy to your existing hosting platform. Inngest invokes your functions via HTTP.",
    },
    {
      title: "Works with serverless functions",
      img: "serverless-queues/middle.png",
      description:
        "Inngest calls your function as events are received. There is no need to set up a worker that polls a queue. Works with your favorite JavaScript framework or any Node.js backend.",
    },
    {
      title: "Automatic retries",
      img: "serverless-queues/right.png",
      description:
        "Failures happen. Inngest retries your functions automatically. The dead letter queue is a thing of the past.",
    },
  ],
  codeSection: {
    title: "Define background jobs in just a few lines of code",
    examples: [
      {
        steps: [
          "Create your function",
          "Declare the event that triggers your function",
          "Define your function steps",
          "Trigger your function with an event",
        ],
        description:
          "Sending events to Inngest automatically triggers background jobs which subscribe to that event.",
        code: `import { inngest } from "./client";

// Instead of sending a welcome email or adding a user to your CRM
// within your signup API endpoint, you can offload to the background:
inngest.createFunction(
  { name: "Post signup flow" },
  { event: "user.signup" },
  async ({ event, step }) => {
    await step.run("Send welcome email", async () => {
      await sendWelcomeEmail({ email: event.data.email });
    });

    await step.run("Add user to CRM", await () => {
      await addUserToHubspot({
        id: event.data.userId,
        email: event.data.email,
      });
    });
  },
);

// Elsewhere in your code, send an event to trigger the function
await inngest.send({
  name: "user.signup",
  data: {
    userId: "6f47ebaa",
    email: "user@example.com",
  }
})`,
      },
    ],
  },
  featureOverflow: [
    {
      title: "Amazing local DX",
      description:
        "Our open source dev server runs on your machine giving you a local sandbox environment with a UI for easy debugging.",
      icon: "WritingFns",
    },
    {
      title: "Full observability and logs",
      description:
        "Check the status of a given job with ease. View your complete event history and function logs anytime.",
      icon: "Tools",
    },
    {
      title: "Fan-out Jobs",
      description:
        "Use a single scheduled function to trigger multiple functions to fan-out logic and run work in parallel.",
      icon: "Server",
    },
    {
      title: "Scheduled Jobs",
      description:
        "Create jobs that sleep or pause for hours, days or weeks to create durable workflows faster than ever before.",
      icon: "Scheduled",
    },
    {
      title: "Retries for max reliability",
      description:
        "Create jobs that sleep or pause for hours, days or weeks to create durable workflows faster than ever before.",
      icon: "Retry",
    },
    {
      title: "TypeScript support",
      description:
        "Define your event payload as TypeScript types to have end-to-end type safety for all your jobs.",
      icon: "SDK",
    },
  ],
  quote: {
    text: "It's sensational - This is the best way to test a background job",
    author: "Garrett Tolbert, Vercel",
  },
  learnMore: {
    description:
      "Dive into our resources and learn how Inngest is the best solution for background tasks or jobs.",
    resources: [
      {
        title: "Quick Start Tutorial",
        description:
          "A step-by-step guide to learn how to build with Inngest in less than 5 minutes.",
        type: "Tutorial",
        href: "/docs/quick-start",
      },
      {
        title: "Running Background Jobs",
        description: "How to background jobs without the queues and workers.",
        type: "Guide",
        href: "/docs/guides/background-jobs",
      },
      {
        title: "Using TypeScript with Inngest",
        description:
          "Learn how our SDK gives you typesafety from sending events to running functions.",
        type: "Docs",
        href: "/docs/typescript",
      },
    ],
  },
};
