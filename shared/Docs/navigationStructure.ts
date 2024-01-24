import {
  CogIcon,
  CommandLineIcon,
  HomeIcon,
  LifebuoyIcon,
  PlayIcon,
  QuestionMarkCircleIcon,
} from "@heroicons/react/24/outline";
import GoIcon from "src/shared/Icons/Go";
import GuideIcon from "src/shared/Icons/Guide";
import PythonIcon from "src/shared/Icons/Python";
import TypeScriptIcon from "src/shared/Icons/TypeScript";
import { StatusIcon } from "src/shared/StatusWidget";

// A basic link in the nav
type NavLink = {
  title: string;
  href: string;
};
// A group nested of nav links with a header
type NavGroup = {
  title: string;
  icon?: React.FC<React.SVGProps<SVGSVGElement>>;
  links: (NavLink | NavSection)[];
};
// A nav section with a nested navigation section
type NavSection = {
  title: string;
  icon?: React.FC<React.SVGProps<SVGSVGElement>>;
  href: string;
  matcher?: RegExp;
  sectionLinks: {
    title: string;
    links: NavLink[];
  }[];
};

const sectionGettingStarted = [
  {
    title: "Quick start tutorials",
    links: [
      {
        title: "Next.js",
        href: "/docs/quick-start",
      },
    ],
  },
  {
    title: "Learn the basics",
    links: [
      { title: "Installing the SDK", href: `/docs/sdk/overview` },
      { title: "Serving the API & Frameworks", href: `/docs/sdk/serve` },
      { title: "Writing Functions", href: `/docs/functions` },
      { title: "Sending Events", href: `/docs/events` },
      {
        title: "Multi-step Functions",
        href: `/docs/functions/multi-step`,
      },
      { title: "Inngest Apps", href: `/docs/apps` },
      {
        title: "Local Development",
        href: `/docs/local-development`,
      },
    ],
  },
];
const sectionGuides = [
  {
    title: "Patterns",
    links: [
      {
        title: "Background jobs",
        href: `/docs/guides/background-jobs`,
      },
      {
        title: "Enqueueing future jobs",
        href: `/docs/guides/enqueueing-future-jobs`,
      },
      {
        title: "Parallelize steps",
        href: `/docs/guides/step-parallelism`,
      },
      {
        title: "Fan-out (one-to-many)",
        href: `/docs/guides/fan-out-jobs`,
      },
      {
        title: "Invoking functions directly",
        href: `/docs/guides/invoking-functions-directly`,
      },
      {
        title: "Sending events from functions",
        href: `/docs/guides/sending-events-from-functions`,
      },
      {
        title: "Batching events",
        href: `/docs/guides/batching`,
      },
      {
        title: "Scheduled functions",
        href: `/docs/guides/scheduled-functions`,
      },
    ],
  },
  {
    title: "How to",
    links: [
      {
        title: "Concurrency",
        href: `/docs/guides/concurrency`,
      },
      {
        title: "Handling idempotency",
        href: `/docs/guides/handling-idempotency`,
      },
      {
        title: "Cancel running functions",
        href: `/docs/guides/cancel-running-functions`,
      },
      {
        title: "Logging",
        href: `/docs/guides/logging`,
      },
      {
        title: "Writing expressions",
        href: `/docs/guides/writing-expressions`,
      },
    ],
  },
  {
    title: "Use cases",
    links: [
      {
        title: "User-defined Workflows",
        href: `/docs/guides/user-defined-workflows`,
      },
      {
        title: "Handling Clerk webhook events",
        href: `/docs/guides/clerk-webhook-events`,
      },
      {
        title: "Trigger code from Retool",
        href: `/docs/guides/trigger-your-code-from-retool`,
      },
      {
        title: "Instrumenting GraphQL",
        href: `/docs/guides/instrumenting-graphql`,
      },
    ],
  },
];

const sectionPlatform = [
  {
    title: "Going to production",
    links: [
      {
        title: "Working with apps",
        href: `/docs/apps/cloud`,
      },
      { title: "Deploy: Vercel", href: `/docs/deploy/vercel` },
      { title: "Deploy: Netlify", href: `/docs/deploy/netlify` },
      {
        title: "Deploy: Cloudflare Pages",
        href: `/docs/deploy/cloudflare`,
      },
    ],
  },
  {
    title: "Inngest Cloud",
    links: [
      {
        title: "Working with environments",
        href: `/docs/platform/environments`,
      },
      {
        title: "Creating an event key",
        href: `/docs/events/creating-an-event-key`,
      },
      {
        title: "Consuming webhook events",
        href: `/docs/platform/webhooks`,
      },
      {
        title: "Replaying functions",
        href: `/docs/platform/replay`,
      },
    ],
  },
  {
    title: "Usage Limits",
    links: [
      {
        title: "Inngest Cloud",
        href: `/docs/usage-limits/inngest`,
      },
      {
        title: "Serverless providers",
        href: `/docs/usage-limits/providers`,
      },
    ],
  },
];

const sectionTypeScriptReference = [
  {
    title: "Overview",
    // TODO - Allow this to be flattened w/ NavGroup
    links: [
      {
        title: "Introduction",
        href: `/docs/reference/typescript`,
      },
    ],
  },
  {
    title: "Inngest Client",
    links: [
      {
        title: "Create the client",
        href: `/docs/reference/client/create`,
      },
    ],
  },
  {
    title: "Functions",
    links: [
      {
        title: "Create function",
        href: `/docs/reference/functions/create`,
      },
      {
        title: "Error handling & retries",
        href: `/docs/functions/retries`,
        // href: `/docs/reference/functions/error-handling`,
      },
      {
        title: "Handling failures",
        href: `/docs/reference/functions/handling-failures`,
      },
      {
        title: "Cancel running functions",
        href: `/docs/functions/cancellation`,
        // href: `/docs/reference/functions/cancel-running-functions`,
      },
      {
        title: "Concurrency",
        href: `/docs/functions/concurrency`,
        // href: `/docs/reference/functions/concurrency`,
      },
      {
        title: "Rate limit",
        href: `/docs/reference/functions/rate-limit`,
      },
      {
        title: "Debounce",
        href: `/docs/reference/functions/debounce`,
      },
      {
        title: "Function run priority",
        href: `/docs/reference/functions/run-priority`,
      },
      // {
      //   title: "Logging",
      //   href: `/docs/reference/functions/logging`,
      // },
      {
        title: "Referencing functions",
        href: `/docs/functions/references`,
      },
    ],
  },
  {
    title: "Steps",
    links: [
      {
        title: "step.run()",
        href: `/docs/reference/functions/step-run`,
        className: "font-mono",
      },
      {
        title: "step.sleep()",
        href: `/docs/reference/functions/step-sleep`,
        className: "font-mono",
      },
      {
        title: "step.sleepUntil()",
        href: `/docs/reference/functions/step-sleep-until`,
        className: "font-mono",
      },
      {
        title: "step.invoke()",
        href: `/docs/reference/functions/step-invoke`,
        className: "font-mono",
      },
      {
        title: "step.waitForEvent()",
        href: `/docs/reference/functions/step-wait-for-event`,
        className: "font-mono",
      },
      {
        title: "step.sendEvent()",
        href: `/docs/reference/functions/step-send-event`,
        className: "font-mono",
      },
    ],
  },
  {
    title: "Events",
    links: [
      {
        title: "Send",
        href: `/docs/reference/events/send`,
      },
    ],
  },
  {
    title: "Serve",
    links: [
      // {
      //   title: "Framework handlers",
      //   href: `/docs/sdk/serve`,
      // },
      {
        title: "Configuration",
        href: `/docs/reference/serve`,
      },
      { title: "Streaming", href: `/docs/streaming` },
    ],
  },
  {
    title: "Middleware",
    links: [
      {
        title: "Overview",
        href: `/docs/reference/middleware/overview`,
      },
      {
        title: "Creating middleware",
        href: `/docs/reference/middleware/create`,
      },
      {
        title: "Lifecycle",
        href: `/docs/reference/middleware/lifecycle`,
      },
      {
        title: "Examples",
        href: `/docs/reference/middleware/examples`,
      },
      {
        title: "TypeScript",
        href: `/docs/reference/middleware/typescript`,
      },
    ],
  },
  {
    title: "Using the SDK",
    links: [
      {
        title: "Environment variables",
        href: `/docs/sdk/environment-variables`,
      },
      {
        title: "Using TypeScript",
        href: `/docs/typescript`,
      },
      {
        title: "ESLint plugin",
        href: `/docs/sdk/eslint`,
      },
      { title: "Upgrading to v3", href: `/docs/sdk/migration` },
    ],
  },
];

const sectionPythonReference = [
  {
    title: "Overview",
    // TODO - Allow this to be flattened w/ NavGroup
    links: [
      {
        title: "Introduction",
        href: `/docs/reference/python`,
      },
      {
        title: "Quick start",
        href: `/docs/reference/python/overview/quick-start`,
      },
      {
        title: "Environment variables",
        href: `/docs/reference/python/overview/env-vars`,
      },
      {
        title: "Production mode",
        href: `/docs/reference/python/overview/prod-mode`,
      },
    ],
  },
  {
    title: "Client",
    links: [
      {
        title: "Overview",
        href: `/docs/reference/python/client/overview`,
      },
      {
        title: "Send events",
        href: `/docs/reference/python/client/send`,
      },
    ],
  },
  {
    title: "Functions",
    links: [
      {
        title: "Create function",
        href: `/docs/reference/python/functions/create`,
      },
    ],
  },
  {
    title: "Steps",
    links: [
      {
        title: "invoke",
        href: `/docs/reference/python/steps/invoke`,
      },
      {
        title: "invoke_by_id",
        href: `/docs/reference/python/steps/invoke_by_id`,
      },
      {
        title: "parallel",
        href: `/docs/reference/python/steps/parallel`,
      },
      {
        title: "run",
        href: `/docs/reference/python/steps/run`,
      },
      {
        title: "send_event",
        href: `/docs/reference/python/steps/send-event`,
      },
      {
        title: "sleep",
        href: `/docs/reference/python/steps/sleep`,
      },
      {
        title: "sleep_until",
        href: `/docs/reference/python/steps/sleep-until`,
      },
      {
        title: "wait_for_event",
        href: `/docs/reference/python/steps/wait-for-event`,
      },
    ],
  },
  {
    title: "Middleware",
    links: [
      {
        title: "Overview",
        href: `/docs/reference/python/middleware/overview`,
      },
    ],
  },
];

export const topLevelNav = [
  {
    title: "Home",
    icon: HomeIcon,
    href: `/docs/`,
  },
  {
    title: "Getting started",
    icon: PlayIcon,
    href: "/docs/quick-start",
    sectionLinks: sectionGettingStarted,
  },
  {
    title: "Guides",
    icon: GuideIcon,
    href: "/docs/guides",
    matcher: /\/guides/,
    sectionLinks: sectionGuides,
  },
  {
    title: "Platform",
    icon: CogIcon,
    href: "/docs/platform",
    matcher: /\/platform/,
    sectionLinks: sectionPlatform,
  },
  {
    title: "SDK Reference",
    links: [
      {
        title: "TypeScript",
        icon: TypeScriptIcon,
        href: `/docs/reference/typescript`,
        sectionLinks: sectionTypeScriptReference,
      },
      {
        title: "Python",
        icon: PythonIcon,
        href: `/docs/reference/python`,
        tag: "Beta",
        sectionLinks: sectionPythonReference,
      },
      {
        title: "Go",
        icon: GoIcon,
        href: `https://pkg.go.dev/github.com/inngest/inngestgo`,
        tag: "Beta",
        target: "_blank",
      },
    ],
  },
  {
    title: "API",
    links: [
      {
        title: "REST API",
        icon: CommandLineIcon,
        href: `https://api-docs.inngest.com/docs/inngest-api`,
        tag: "In progress",
        target: "_blank",
      },
    ],
  },
  {
    title: "Help",
    links: [
      {
        title: "FAQs",
        icon: QuestionMarkCircleIcon,
        href: `/docs/faq`,
      },
      {
        title: "Status Page",
        icon: StatusIcon,
        href: "https://status.inngest.com",
      },
      {
        title: "Support Center",
        icon: LifebuoyIcon,
        href: "https://app.inngest.com/support",
      },
    ],
  },
];
