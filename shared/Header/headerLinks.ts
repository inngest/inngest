import IconBackgroundTasks from "./Icons/IconBackgroundTasks";
import IconDeploying from "./Icons/IconDeploying";
import IconDocs from "./Icons/IconDocs";
import IconFunctions from "./Icons/IconFunctions";
import IconJourney from "./Icons/IconJourney";
import IconPatterns from "./Icons/IconPatterns";
import IconScheduled from "./Icons/IconScheduled";
import IconSendEvents from "./Icons/IconSendEvents";
import IconSteps from "./Icons/IconSteps";
import IconTools from "./Icons/IconTools";
import IconWritingFns from "./Icons/IconWritingFns";


const productLinks = {
  featuredTitle: "Product",
  featured: [
    {
      title: "TypeScript & JavaScript SDK",
      desc: "Event-driven and and scheduled serverless functions",
      url: "/features/sdk?ref=nav",
      icon: IconFunctions,
      iconBg: "bg-indigo-500",
    },
    {
      title: "Step Functions",
      desc: "Build complex conditional workflows",
      url: "/features/step-functions?ref=nav",
      icon: IconSteps,
      iconBg: "bg-violet-500",
    },
  ],
  linksTitle: "Use Cases",
  links: [
    {
      title: "Scheduled & cron jobs",
      url: "/uses/serverless-cron-jobs?ref=nav",
      icon: IconScheduled,
    },
    {
      title: "Background tasks",
      url: "/uses/serverless-node-background-jobs?ref=nav",
      icon: IconBackgroundTasks,
    },
    {
      title: "Internal tools",
      url: "/uses/internal-tools?ref=nav",
      icon: IconTools,
    },
    {
      title: "User journey automation",
      url: "/uses/user-journey-automation?ref=nav",
      icon: IconJourney,
    },
  ]
};

const learnLinks = {
  featuredTitle: "Learn",
  featured: [
    {
      title: "Docs",
      desc: "Everything you need to know about our event-driven platform",
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
  ],
  linksTitle: "Quick Starts",
  links: [
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
  ]
};

export {
  productLinks,
  learnLinks
};