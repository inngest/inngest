import {
  IconBackgroundTasks,
  IconDeploying,
  IconDocs,
  IconSDK,
  IconJourney,
  IconPatterns,
  IconScheduled,
  IconSendEvents,
  IconSteps,
  IconTools,
  IconWritingFns,
  IconCompiling,
  IconPower,
  IconRetry,
} from "../Icons/duotone";

const productLinks = {
  featuredTitle: "Product",
  featured: [
    {
      title: "How Inngest Works",
      desc: "Learn about the Inngest serverless queue & workflow engine",
      url: "/product/how-inngest-works?ref=nav",
      icon: IconSteps,
      iconBg: "bg-indigo-500",
    },
  ],
  linksTitle: "Use Cases",
  linksTheme: "indigo",
  links: [
    {
      title: "AI + LLMs",
      url: "/ai?ref=nav",
      icon: IconSDK,
    },
    {
      title: "Durable workflows",
      url: "/uses/durable-workflows?ref=nav",
      icon: IconRetry,
    },
    {
      title: "Workflow engines",
      url: "/uses/workflow-engine?ref=nav",
      icon: IconJourney,
    },
    {
      title: "Serverless queues",
      url: "/uses/serverless-queues?ref=nav",
      icon: IconSteps,
    },
    {
      title: "Background jobs",
      url: "/uses/serverless-node-background-jobs?ref=nav",
      icon: IconBackgroundTasks,
    },
    {
      title: "Scheduled & cron jobs",
      url: "/uses/serverless-cron-jobs?ref=nav",
      icon: IconScheduled,
    },
  ],
};

const learnLinks = {
  featuredTitle: "Documentation",
  featured: [
    {
      title: "Docs",
      desc: "SDK and platform guides and references",
      url: "/docs?ref=nav",
      icon: IconDocs,
      iconBg: "bg-blue-500",
    },
    // {
    //   title: "Patterns: Async & event-driven",
    //   desc: "How to build asynchronous functionality by example",
    //   url: "/patterns?ref=nav",
    //   icon: IconPatterns,
    //   iconBg: "bg-sky-500",
    // },
  ],
  linksTitle: "Guides",
  linksTheme: "blue",
  links: [
    {
      title: "Quick start tutorial",
      url: "/docs/quick-start?ref=nav",
      icon: IconCompiling,
    },
    {
      title: "Concurrency",
      url: "/docs/guides/concurrency?ref=nav",
      icon: IconWritingFns,
    },
    {
      title: "Handling Idempotency",
      url: "/docs/guides/handling-idempotency?ref=nav",
      icon: IconSendEvents,
    },
    {
      title: "Going to production",
      url: "/docs/deploy?ref=nav",
      icon: IconDeploying,
    },
  ],
};

export { productLinks, learnLinks };
