import { GetStaticProps, GetStaticPaths } from "next";
import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";
import Footer from "src/shared/Footer";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import PageHeader from "src/shared/PageHeader";
import SectionHeader from "src/shared/SectionHeader";
import Learning from "src/shared/Cards/Learning";
import PageContainer from "src/shared/layout/PageContainer";
import Image from "next/image";

import {
  IconRetry,
  IconServer,
  IconTools,
  IconUnlock,
  IconWritingFns,
  IconProps,
} from "../../shared/Icons/duotone";

const Icons: { [key: string]: React.FC<IconProps> } = {
  Retry: IconRetry,
  Server: IconServer,
  Tools: IconTools,
  Unlock: IconUnlock,
  WritingFns: IconWritingFns,
};

type IconType = keyof typeof Icons;
export type UseCase = {
  title: string;
  lede: string;
  keyFeatures: {
    title: string;
    img: string;
    description: string;
  }[];
  code: string;
  featureOverflow: {
    title: string;
    description: string;
    icon: IconType;
  }[];
  quote: {
    text: string;
    author: string;
  };
  learning: {
    title: string;
    description: string;
    type: string;
    href: string;
  }[];
};

export const getStaticProps: GetStaticProps = async (ctx) => {
  const { data } = require(`./cases/${ctx.params.case}.ts`);
  const stringData = JSON.stringify(data);
  return {
    props: {
      stringData,
      designVersion: "2",
    },
  };
};

export const getStaticPaths: GetStaticPaths = async () => {
  const fs = require("node:fs");
  const fileNames = fs.readdirSync("./pages/uses/cases");

  const paths = fileNames.map((fileName) => {
    return {
      params: {
        case: fileName.replace(/\.ts$/, ""),
      },
    };
  });

  return {
    paths,
    fallback: false,
  };
};

const data = {
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
      icon: IconWritingFns,
    },
    {
      title: "Full observability and logs",
      description:
        "Check the status of a given job with ease. View your complete event history and function logs anytime.",
      icon: IconTools,
    },
    {
      title: "Fan-out Jobs",
      description:
        "Events can trigger multiple functions, meaning that you can separate logic into different jobs that consume the same event.",
      icon: IconServer,
    },
    {
      title: "Delays",
      description:
        "Use TypeScript to build, test, and deploy serverless functions driven by  events or a schedule to any platform in sections, with zero infrastructure.",
      icon: IconRetry,
    },
    {
      title: "Open Source",
      description:
        "Learn how Inngest works, or self-host if you prefer to manage it yourself.",
      icon: IconUnlock,
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

export default function useCase({ stringData }) {
  const data: UseCase = JSON.parse(stringData);
  return (
    <PageContainer>
      <Header />

      <Container className="my-48">
        <PageHeader title={data.title} lede={data.lede} />
      </Container>

      <Container>
        <SectionHeader title="Key Features" />
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 mt-8 gap-2">
          {data.keyFeatures.map((feature, i) => (
            <div
              key={i}
              className="max-w-[600px] m-auto md:m-0 bg-slate-950/80 overflow-hidden rounded-lg border-slate-900/10"
            >
              <Image
                className="rounded-t-lg lg:rounded-t-none lg:rounded-r-lg group-hover:rounded-lg"
                src={`/assets/use-cases/${feature.img}`}
                width={600}
                height={340}
                quality={95}
              />
              <div className="p-6 lg:p-10">
                <h3 className="text-lg lg:text-xl text-white mb-2.5">
                  {feature.title}
                </h3>
                <p className="text-sm text-indigo-200 leading-6">
                  {feature.description}
                </p>
              </div>
            </div>
          ))}
        </div>
      </Container>

      <Container className=" my-40">
        <SectionHeader
          title="Queue work"
          lede="Queue work in just a few lines of code with Inngest."
        />
        <div className="flex mt-16 flex-col lg:flex-row flex-start ">
          <div className="text-slate-200 mb-10 lg:mb-0 lg:pr-20 max-w-[400px] justify-center flex flex-col gap-3">
            <p className="flex items-center gap-3">
              <span className="bg-slate-800 rounded flex items-center justify-center w-6 h-6 text-xs font-bold">
                1
              </span>{" "}
              Define your event payload type
            </p>
            <p className="flex items-center gap-3">
              <span className="bg-slate-800 rounded flex items-center justify-center w-6 h-6 text-xs font-bold">
                2
              </span>{" "}
              Send events with type{" "}
            </p>
            <p className="flex items-center gap-3">
              <span className="bg-slate-800 rounded flex items-center justify-center w-6 h-6 text-xs font-bold">
                3
              </span>{" "}
              Define your functions with that
            </p>
            <p className="text-sm text-slate-300 mt-4">
              Functions trigger as events are received. Inngest calls all
              matching functions via HTTP.
            </p>
          </div>
          <SyntaxHighlighter
            language="javascript"
            showLineNumbers={false}
            style={syntaxThemeDark}
            codeTagProps={{ className: "code-window" }}
            // className="hello"
            customStyle={{
              fontSize: "0.8rem",
              padding: "1.5rem",
              backgroundColor: "#0C1323",
              display: "inline-flex",
            }}
          >
            {data.code}
          </SyntaxHighlighter>
        </div>
      </Container>

      <Container className="my-40">
        <SectionHeader
          title="Everything you need to build"
          lede="Inngest is the easiest way to build scheduled jobs in your app, no matter what framework or platform you use."
        />
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8 xl:gap-16 mt-20">
          {data.featureOverflow.map((feature, i) => (
            <div key={i}>
              <h3 className="text-slate-50 text-lg lg:text-xl mb-2 flex items-center gap-1 -ml-2">
                {feature.icon && (
                  <Icon name={feature.icon} size={30} color="indigo" />
                )}
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

const Icon = ({ name, ...props }: { name: IconType } & IconProps) => {
  const C = Icons[name];
  return <C {...props} />;
};
