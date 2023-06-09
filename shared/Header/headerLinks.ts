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
    // {
    //   title: "Step Functions",
    //   desc: "Build complex conditional workflows",
    //   url: "/features/step-functions?ref=nav",
    //   icon: IconSteps,
    //   iconBg: "bg-violet-500",
    // },
  ],
  linksTitle: "Use Cases",
  linksTheme: "indigo",
  links: [
    {
      title: "Serverless Queues",
      url: "/uses/serverless-queues?ref=nav",
      icon: IconSteps,
    },
    // {
    //   title: "Durable Functions",
    //   url: "/uses/durable-functions?ref=nav",
    //   icon: IconScheduled,
    // },
    {
      title: "Background Jobs",
      url: "/uses/serverless-node-background-jobs?ref=nav",
      icon: IconBackgroundTasks,
    },
    {
      title: "AI + LLMs",
      url: "/uses/zero-infra-llm-ai?ref=nav",
      icon: IconSDK,
    },
    {
      title: "Scheduled & cron jobs",
      url: "/uses/serverless-cron-jobs?ref=nav",
      icon: IconScheduled,
    },
    // {
    //   title: "Complex Workflows",
    //   url: "/uses/complex-workflows?ref=nav",
    //   icon: IconJourney,
    // },
    //
    // {
    //   title: "User journey automation",
    //   url: "/uses/user-journey-automation?ref=nav",
    //   icon: IconJourney,
    // },
  ],
};

const learnLinks = {
  featuredTitle: "Learn",
  featured: [
    {
      title: "Docs",
      desc: "SDK and platform guides and references",
      url: "/docs?ref=nav",
      icon: IconDocs,
      iconBg: "bg-blue-500",
    },
    {
      title: "Patterns: Async & event-driven",
      desc: "How to build asynchronous functionality by example",
      url: "/patterns?ref=nav",
      icon: IconPatterns,
      iconBg: "bg-sky-500",
    },
    {
      title: "AI-Personalized Documentation",
      desc: "Learn how to use Inngest with personalized code examples",
      url: "/ai-personalized-documentation?ref=nav",
      icon: () => "✨",
      iconBg: "bg-indigo-900/50 text-[20px]",
    },
  ],
  linksTitle: "Quick Starts",
  linksTheme: "blue",
  links: [
    {
      title: "Quick start tutorial",
      url: "/docs/quick-start?ref=nav",
      icon: IconCompiling,
    },
    {
      title: "Writing functions",
      url: "/docs/functions?ref=nav",
      icon: IconWritingFns,
    },
    {
      title: "Sending events",
      url: "/docs/events?ref=nav",
      icon: IconSendEvents,
    },
    {
      title: "Deploying",
      url: "/docs/deploy?ref=nav",
      icon: IconDeploying,
    },
  ],
};

export { productLinks, learnLinks };
