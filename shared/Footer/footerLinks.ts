import {
  IconSDK,
  IconSteps,
  IconDocs,
  IconPatterns,
  IconDeploying,
  IconScheduled,
  IconBackgroundTasks,
  IconTools,
  IconJourney,
} from "../Icons/duotone";

const footerLinks = [
  {
    name: "Product",
    links: [
      {
        label: "Function SDK",
        url: "/features/sdk?ref=footer",
        icon: IconSDK,
      },
      {
        label: "Step Functions",
        url: "/features/step-functions?ref=footer",
        icon: IconSteps,
      },
      {
        label: "Documentation",
        url: "/docs?ref=footer",
        icon: IconDocs,
      },
      {
        label: "Patterns: Async + Event-Driven",
        url: "/patterns?ref=footer",
        icon: IconPatterns,
      },
      {
        label: "Self Hosting",
        url: "/docs/self-hosting?ref=footer",
        icon: IconDeploying,
      },
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
        label: "Node.js background jobs",
        url: "/uses/serverless-node-background-jobs?ref=footer",
        icon: IconBackgroundTasks,
      },
      {
        label: "Internal tools",
        url: "/uses/internal-tools?ref=footer",
        icon: IconTools,
      },
      {
        label: "User Journey Automation",
        url: "/uses/user-journey-automation?ref=footer",
        icon: IconJourney,
      },
    ],
  },
  {
    name: "Company",
    links: [
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
    ],
  },
];

export default footerLinks;
