import type { UseCase } from "../../pages/uses/[case]";

export const data: UseCase = {
  title: "Serverless scheduled & cron jobs",
  lede: "Easily create scheduled serverless functions or schedule work for the future in just a few lines of code.",
  heroImage: "/assets/use-cases/serverless-cron-jobs/hero-image.png",
  keyFeatures: [
    {
      title: "Run your function on a schedule",
      description:
        "Easily define your function with a cron schedule and Inngest will automatically invoke your function on that schedule.",
    },
    {
      title: "Run code at a specific time",
      description:
        "Using Inngest's <code>step.sleepUntil</code> to delay running code to an exact timestamp. Schedule any work based off user input.",
    },
    {
      title: "Automatic retries",
      description:
        "Failures happen. Inngest retries your functions automatically.",
    },
  ],
  codeSection: {
    title: "Schedule work in a few lines of code",
    steps: [
      "Define a function using a cron schedule",
      "Or define a function triggered by an event and use step.sleepUntil to delay work until a given timestamp",
    ],
    description:
      "Use both depending on if your schedule work is periodic or dynamic.",
    code: `import { inngest } from "./client";

// Define a function to run on a cron-schedule:
inngest.createFunction(
  { name: "Send weekly digest email" },
  { cron: "TZ=America/New_York 0 9 * * MON " },
  async () => {
    // This function will run every Monday at 9am New York time
  },
);

// Or define a function which sleeps until a given timestamp:
inngest.createFunction(
  { name: "Post Slack reminder" },
  { event: "slack.reminder.scheduled" },
  async ({ event, step }) => {
    await step.sleepUntil(event.data.reminderTimestamp);
    await step.run("Send Slack notification", async () => {
      // This will run after the given reminder timestamp
    })
  },
);
`,
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
  ],
  quote: {
    text: "It's sensational - This is the best way to test a background job",
    author: "Garrett Tolbert, Vercel",
  },
  learnMore: {
    description:
      "Dive into our resources and learn how Inngest is the best solution for serverless scheduled jobs.",
    resources: [
      {
        title: "Scheduled functions with cron",
        description:
          "How to create a schedule function using a crontab syntax.",
        type: "Guide",
        href: "/docs/guides/scheduled-functions",
      },
      {
        title: "Enqueue future jobs",
        description: "How to schedule your code to run at a specific time.",
        type: "Guide",
        href: "/docs/guides/enqueueing-future-jobs",
      },
      {
        title: "Writing scheduled functions",
        description: "Learn how to define scheduled functions.",
        type: "Docs",
        href: "/docs/functions#writing-a-scheduled-function",
      },
    ],
  },
};
