import {
  IconSDK,
  IconDocs,
  IconPatterns,
  IconScheduled,
  IconBackgroundTasks,
  IconJourney,
} from "../Icons/duotone";

const footerLinks = [
  {
    name: "Product",
    links: [
      // {
      //   label: "Function SDK",
      //   url: "/features/sdk?ref=footer",
      //   icon: IconSDK,
      // },
      // {
      //   label: "Step Functions",
      //   url: "/features/step-functions?ref=footer",
      //   icon: IconSteps,
      // },
      {
        label: "Documentation",
        url: "/docs?ref=footer",
        icon: IconDocs,
      },
      {
        label: "Patterns: Async + Event-Driven",
        url: "/patterns?ref=footer",
        icon: IconPatterns,
      }
    ],
  },
  {
    name: "Use Cases",
    links: [
      {
        label: "Serverless queues for TypeScript",
        url: "/uses/serverless-queues?ref=footer",
        icon: IconJourney,
      },
      {
        label: "Scheduled & cron jobs",
        url: "/uses/serverless-cron-jobs?ref=footer",
        icon: IconScheduled,
      },
      {
        label: "AI + LLMs",
        url: "/ai?ref=footer",
        icon: IconSDK,
      },
      {
        label: "Node.js background jobs",
        url: "/uses/serverless-node-background-jobs?ref=footer",
        icon: IconBackgroundTasks,
      },
    ],
  },
  {
    name: "Company",
    links: [
      {
        label: "Roadmap",
        url: "https://roadmap.inngest.com/roadmap?ref=footer",
      },
      {
        label: "Changelog",
        url: "https://roadmap.inngest.com/changelog?ref=footer",
      },
      {
        label: "About",
        url: "/about?ref=footer",
      },
      {
        label: "Careers",
        url: "/careers?ref=footer",
      },
      {
        label: "Blog",
        url: "/blog?ref=footer",
      },
      {
        label: "Contact Us",
        url: "/contact?ref=footer",
      },
      {
        label: "Support",
        url: process.env.NEXT_PUBLIC_SUPPORT_URL,
      },
    ],
  },
];

export default footerLinks;
