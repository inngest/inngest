import type { UseCase } from "../[case]";

export const data: UseCase = {
  title: "Serverless queues for TypeScript",
  lede: "Use Inngest’s type safe SDK to enqueue jobs using events. No polling - Inngest calls your serverless functions.",
  keyFeatures: [
    {
      title: "Nothing to configure",
      img: "serverless-queues/left.png",
      description:
        "Inngest is serverless, and there’s no queue to configure. Just start sending events, and your functions declare which events trigger them.",
    },
    {
      title: "We call your function",
      img: "serverless-queues/middle.png",
      description:
        "Inngest calls your function as events are received. There is no need to set up a worker that polls a queue.",
    },
    {
      title: "Automatic retries",
      img: "serverless-queues/right.png",
      description:
        "Failures happen. Inngest retries your functions automatically. The dead letter queue is a thing of the past.",
    },
  ],
  code: `// Define your event payload with our standard name & date fields
type MyEventType = {
	name: "my.event",
  data: {
    userId: string
  }
}

// Send events to Inngest
inngest.send<MyEventType>({
	name: "my.event", data: { userId: "12345" }
});

// Define your function to handle that event
createFunction<MyEventType>("My handler", "my.event", ({ event }) => {
  // Handle your event
});
`,
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
        "Events can trigger multiple functions, meaning that you can separate logic into different jobs that consume the same event.",
      icon: "Server",
    },
    {
      title: "Delays",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      icon: "Retry",
    },
    {
      title: "Open Source",
      description:
        "Learn how Inngest works, or self-host if you prefer to manage it yourself.",
      icon: "Unlock",
    },
  ],
  quote: {
    text: "A quote from a happy customer about how Inngest is the best event-driven system out there.",
    author: "Someone important, cool company",
  },
  learning: [
    {
      title: "Serverless Queues for Next.js",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      type: "Guide",
      href: "/docs/getting-started",
    },
    {
      title: "Use TypeScript with Inngest",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      type: "Tutorial",
      href: "/docs/getting-started",
    },
    {
      title: "Running Background Jobs",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      type: "Pattern",
      href: "/docs/getting-started",
    },
    {
      title: "Serverless Queues for Next.js",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      type: "Docs",
      href: "/docs/getting-started",
    },
    {
      title: "Use TypeScript with Inngest",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      type: "Blog",
      href: "/docs/getting-started",
    },
  ],
};
