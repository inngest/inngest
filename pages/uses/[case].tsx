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
    type: "Docs" | "Tutorial" | "Guide" | "Pattern" | "Blog";
    href: string;
  }[];
};

export const getStaticProps: GetStaticProps = async (ctx) => {
  const { data } = require(`../../data/uses/${ctx.params.case}.ts`);
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
  const fileNames = fs.readdirSync("./data/uses");

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

export default function useCase({ stringData }) {
  const data: UseCase = JSON.parse(stringData);
  return (
    <PageContainer>
      <Header />

      <Container className="my-48">
        <PageHeader title={data.title} lede={data.lede} />
      </Container>

      <Container>
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 mt-8 gap-2">
          {data.keyFeatures.map((feature, i) => (
            <div
              key={i}
              className="max-w-[600px] m-auto md:m-0 bg-slate-950/80 overflow-hidden rounded-lg border-slate-900/10"
            >
              <Image
                alt={`Graphic of ${feature.title}`}
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
        <SectionHeader title="Queue work in just a few lines of code" />
        <div className="flex mt-16 flex-col lg:flex-row flex-start ">
          <div className="text-slate-200 mb-10 lg:mb-0 lg:pr-20 max-w-[400px] justify-center flex flex-col gap-3">
            <p className="flex items-start gap-3">
              <span className="bg-slate-800 rounded flex items-center justify-center w-6 h-6 text-xs font-bold shrink-0">
                1
              </span>{" "}
              Define your event payload type
            </p>
            <p className="flex items-start gap-3">
              <span className="bg-slate-800 rounded flex items-center justify-center w-6 h-6 text-xs font-bold shrink-0">
                2
              </span>{" "}
              Send events with type{" "}
            </p>
            <p className="flex items-start gap-3">
              <span className="bg-slate-800 rounded flex items-center justify-center w-6 h-6 text-xs font-bold shrink-0">
                3
              </span>{" "}
              Define your functions with that event trigger
            </p>
            <p className="text-sm text-slate-300 mt-4">
              Functions trigger as events are received. Inngest calls all
              matching functions via HTTP.
            </p>
          </div>
          <SyntaxHighlighter
            language="typescript"
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
        <SectionHeader title="Everything you need to build" />
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
          lede="Dive into our resources and learn how Inngest is the best solution for serverless queues for TypeScript."
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
