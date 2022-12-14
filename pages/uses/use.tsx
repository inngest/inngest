import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";
import Footer from "src/shared/Footer";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import PageHeader from "src/shared/PageHeader";
import SectionHeader from "src/shared/SectionHeader";
import Learning from "src/shared/Cards/Learning";
import PageContainer from "src/shared/layout/PageContainer";

import {
  IconSteps,
  IconGuide,
  IconSDK,
  IconTutorial,
  IconCompiling,
  IconScheduled,
  IconBackgroundTasks,
  IconTools,
  IconJourney,
  IconWritingFns,
  IconSendEvents,
  IconDeploying,
  IconDocs,
  IconPatterns,
  IconBlog,
  IconPower,
  IconFiles,
  IconCloud,
  IconServer,
  IconRetry,
} from "../../shared/Icons/duotone";

export async function getStaticProps() {
  return {
    props: {
      designVersion: "2",
    },
  };
}

const data = {
  title: "Serverless queues for TypeScript",
  lede: "Use Inngest’s type safe SDK to enqueue jobs using events. No polling - Inngest calls your serverless functions.",
  keyFeatures: [
    {
      title: "Nothing to configure",
      description:
        "Inngest is serverless, and there’s no queue to configure. Just start sending events, and your functions declare which events trigger them.",
    },
    {
      title: "We call your function",
      description:
        "Inngest calls your function as events are received. There is no need to set up a worker that polls a queue.",
    },
    {
      title: "Automatic retries",
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
      icon: "",
    },
    {
      title: "Full observability and logs",
      description:
        "Check the status of a given job with ease. View your complete event history and function logs anytime.",
      icon: "",
    },
    {
      title: "Fan-out Jobs",
      description:
        "Events can trigger multiple functions, meaning that you can separate logic into different jobs that consume the same event.",
      icon: "",
    },
    {
      title: "Delays",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      icon: "",
    },
    {
      title: "Open Source",
      description:
        "Learn how Inngest works, or self-host if you prefer to manage it yourself.",
      icon: "",
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
      type: "Docs",
      href: "/docs/getting-started",
    },
    {
      title: "Use TypeScript with Inngest",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      type: "Guide",
      href: "/docs/getting-started",
    },
    {
      title: "Running Background Jobs",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      type: "Pattern",
      href: "/docs/getting-started",
    },
  ],
};

export default function template() {
  return (
    <PageContainer>
      <Header />

      <Container className="my-48">
        <PageHeader title={data.title} lede={data.lede} />
      </Container>

      <Container>
        <SectionHeader title="Key Features" />
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-2">
          {data.keyFeatures.map((feature, i) => (
            <div key={i} className="bg-slate-900/90 p-6 lg:p-8 rounded">
              <h3 className="text-lg lg:text-xl text-white mb-2.5">
                {feature.title}
              </h3>
              <p className="text-sm text-indigo-200 leading-6">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </Container>

      <Container className=" my-40">
        <SectionHeader
          title="Queue work"
          lede="Queue work in just a few lines of code with Inngest."
        />

        <SyntaxHighlighter
          language="javascript"
          showLineNumbers={false}
          style={syntaxThemeDark}
          codeTagProps={{ className: "code-window" }}
          // className="hello"
          customStyle={{
            backgroundColor: "transparent",
            fontSize: "0.8rem",
            padding: "1.5rem",
          }}
        >
          {data.code}
        </SyntaxHighlighter>
      </Container>

      <Container className="my-40">
        <SectionHeader
          title="Everything you need to build"
          lede="Inngest is the easiest way to build scheduled jobs in your app, no matter what framework or platform you use."
        />
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8 xl:gap-16 mt-20">
          {data.featureOverflow.map((feature, i) => (
            <div key={i}>
              <h3 className="text-slate-50 text-lg lg:text-xl mb-2">
                {feature.title}
              </h3>
              <p className="text-indigo-200 text-sm leading-loose">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </Container>

      <Container className="flex flex-col items-center gap-4 my-48">
        <h3 className="text-white text-center text-xl max-w-xl">
          "{data.quote.text}"
        </h3>
        <p className="text-indigo-200">{data.quote.author}</p>
      </Container>

      <Container>
        <SectionHeader
          title="Learn more"
          lede="Add Inngest to your stack in a few lines of code, then deploy to your existing provider. You don’t have to change anything to get started."
        />
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 mt-16">
          {data.learning.map((learningItem, i) => (
            <Learning
              key={i}
              href={learningItem.href}
              title={learningItem.title}
              description={learningItem.description}
              type={learningItem.type}
            />
          ))}
        </div>
      </Container>
      <Footer />
    </PageContainer>
  );
}
