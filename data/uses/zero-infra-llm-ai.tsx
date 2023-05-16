import type { UseCase } from "../../pages/uses/[case]";

export const data: UseCase = {
  title: "Zero-infrastructure LLM & AI",
  lede: "Build LLM and AI chains reliably in minutes — no memory, state, or infrastructure needed. Locally test then deploy to any platform using normal code.",
  keyFeatures: [
    {
      title: "Automatic Memory & Context",
      img: "serverless-queues/left.png",
      description:
        "Functions automatically maintain state, allowing you to reference the output of any API call in normal code without using databases or caching.",
    },
    {
      title: "Fully Serverless",
      img: "serverless-queues/middle.png",
      description:
        "Deploy to any provider, on any platform. Inngest ensures that each step is called once, and spreads each step over multiple function invocations while maintaining state.",
    },
    {
      title: "Reliable by Default",
      img: "serverless-queues/right.png",
      description:
        "Inngest automatically retries steps within functions on error. Never worry about issues with your provider's availability or API downtime again.",
    },
  ],
  codeSection: {
    title: "Build reliable AI products in a few lines of code",
    examples: [
      {
        title: "Chained LLMs",
        steps: [
          "Define an event to trigger your chain function",
          "Use step.run for reliable API calls",
          "Return state from each step",
          "Use state in following steps in your chain",
        ],
        description:
          "Automatic retries and persisted state across all steps in your chain.",
        code: `import { inngest } from "./client";

inngest.createFunction(
  { name: "Summarize chat and documents" },
  { event: "api/chat.submitted" },
  async ({ event, step }) => {
    const llm = new OpenAI();

    const output = await step.run("Summarize input", async () => {
      return await llm.createCompletion({
        model: "gpt-3.5-turbo",
        prompt: createSummaryPrompt(event.data.input),
      });
    });

    const title = await step.run("Generate a title", async () => {
      return await llm.createCompletion({
        model: "gpt-3.5-turbo",
        prompt: createTitlePrompt(output),
      });
    });

    await step.run("Save to DB", async () => {
      await db.summaries.create({ output, title, requestID: event.data.requestID });
    });

    return { output, title };
  },
);`,
      },
    ],
  },
  featureOverflowTitle: "Advanced features, for production-ready systems",
  featureOverflow: [
    {
      title: "Cancellation",
      description:
        "Cancel long running functions automatically or via an API call, keeping your resources free.",
      icon: "Compiling",
    },
    {
      title: "Concurrency",
      description:
        "Set custom concurrency limits on functions or specific API calls, and only run when there's capacity.",
      icon: "Steps",
    },
    {
      title: "Per-User Rate-Limiting",
      description:
        "Set hard rate limits on functions using custom keys like user IDs, ensuring that you use your model tokens or GPU efficiently.",
      icon: "Server",
    },
    // {
    //   title: "Scheduled work for later",
    //   description:
    //     "Create jobs that sleep or pause for hours, days or weeks to create durable workflows faster than ever before.",
    //   icon: "Scheduled",
    // },
    // {
    //   title: "TypeScript support",
    //   description:
    //     "Define your event payload as TypeScript types to have end-to-end type safety for all your jobs.",
    //   icon: "SDK",
    // },
  ],
  // quote: {
  //   text: "It's sensational - This is the best way to test a background job",
  //   author: "Garrett Tolbert, Vercel",
  // },
  learnMore: {
    description:
      "Dive into our resources and learn how Inngest is the best solution for building reliable LLM + AI products in production.",
    resources: [
      {
        title: "Running chained LLMs with TypeScript in production",
        description: "What is chaining and when should you use it?",
        type: "Blog",
        href: "/blog/running-chained-llms-typescript-in-production",
      },
      // TODO - Add upcoming guide here:
      // {
      //   title: "LLM Chaining",
      //   description: "How to use OpenAI with Inngest steps to chain prompts and persist state.",
      //   type: "Guide",
      //   href: "/docs/guides/????????????",
      // },
      // {
      //   title: "Using TypeScript with Inngest",
      //   description:
      //     "Learn how our SDK gives you typesafety from sending events to running functions.",
      //   type: "Docs",
      //   href: "/docs/typescript",
      // },
    ],
  },
};
