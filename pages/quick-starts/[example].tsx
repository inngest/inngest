import styled from "@emotion/styled";
import shuffle from "lodash.shuffle";
import { marked } from "marked";
import { GetStaticPaths, GetStaticProps } from "next";
import Link from "next/link";
import Script from "next/script";
import { useEffect, useMemo, useState } from "react";
import Button from "src/shared/legacy/Button";
import { CommandSnippet } from "src/shared/legacy/CommandSnippet";
import Footer from "src/shared/legacy/Footer";
import Github from "src/shared/Icons/Github";
import Nav from "src/shared/legacy/nav";
import { reqWithSchema } from "src/utils/fetch";
import { blobSchema, getExamples } from ".";
import hljs from "highlight.js";

interface Props {
  id: string;
  name: string;
  description?: string;
  readme?: string;
  examples: Awaited<ReturnType<typeof getExamples>>;
}

export default function LibraryExamplePage(props: Props) {
  const [loadingMermaid, setLoadingMermaid] = useState(true);

  useEffect(() => {
    // @ts-ignore
    if (typeof mermaid !== "undefined") {
      // @ts-ignore
      mermaid?.contentLoaded();
    }
  }, [loadingMermaid, props.readme]);

  const nextExamples = useMemo(() => {
    return shuffle(props.examples)
      .filter(({ id }) => id !== props.id)
      .slice(0, 3);
  }, [props.examples]);

  return (
    <div>
      <Script
        src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"
        onLoad={() => {
          // Ignoring as we don't have (or really need) global typing for
          // mermaid.
          // @ts-ignore
          mermaid.initialize({ startOnLoad: true });

          // Set this to force a re-render
          setLoadingMermaid(false);
        }}
      />
      <Nav sticky nodemo />
      <div className="container mx-auto pt-32 pb-24 flex flex-row">
        <div className="text-center px-6 max-w-4xl mx-auto flex flex-col space-y-6">
          <h1>{props.name}</h1>
          <p className="subheading">{props.description}</p>
          <div>
            <CommandSnippet
              command={`npx inngest-cli init --template github.com/inngest/inngest#examples/${props.id}`}
              copy
            />
          </div>
          <div className="flex flex-col justify-center space-y-4 sm:flex-row sm:space-y-0 sm:space-x-4">
            <Button
              kind="primary"
              href={`/quick-starts?ref=quick-starts/${props.id}`}
            >
              See more quickstarts
            </Button>
            <Button
              kind="outline"
              href={`https://github.com/inngest/inngest/tree/main/examples/${props.id}`}
              target="_blank"
              className="flex flex-row items-center justify-center space-x-1 !ml-0 sm:!ml-4"
            >
              <Github />
              <div>Explore the code</div>
            </Button>
          </div>
        </div>
      </div>
      {props.readme ? (
        <div className="mx-auto max-w-2xl pb-24 flex items-center justify-center">
          <div
            className="prose px-6 prose-a:text-[#5d5fef] max-w-none w-full"
            dangerouslySetInnerHTML={{ __html: props.readme }}
          />
        </div>
      ) : null}
      {nextExamples.length ? (
        <Highlights>
          <div className="container mx-auto p-12">
            <h2>More quickstarts</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-12 mt-4">
              {nextExamples.map((example) => (
                <div key={example.id} className="w-full h-full flex flex-col">
                  <Link
                    key={example.id}
                    href={`/quick-starts/${example.id}?ref=quick-starts/${props.id}`}
                    className="rounded-lg border border-gray-200 p-6 flex flex-col space-y-2 bg-black transition-all transform hover:scale-105 hover:shadow-lg flex-1"
                  >
                    <div className="font semi-bold text-white">
                      {example.name}
                    </div>
                    {example.description ? (
                      <div className="font-xs text-gray-200">
                        {example.description}
                      </div>
                    ) : null}
                    <div className="flex-1" />
                    <span className="text-blue-500 font-semibold text-right">
                      Explore →
                    </span>
                  </Link>
                </div>
              ))}
            </div>
            <div className="w-full flex items-center justify-center mt-10">
              <Button
                kind="primary"
                href={`/quick-starts?ref=quick-starts/${props.id}`}
              >
                See all quickstarts
              </Button>
            </div>
          </div>
        </Highlights>
      ) : props.readme ? (
        <div className="w-full flex items-center justify-center">
          <Button
            kind="primary"
            href={`/quick-starts?ref=quick-starts/${props.id}`}
          >
            See all quickstarts
          </Button>
        </div>
      ) : null}
      <Footer />
    </div>
  );
}

export const getStaticPaths: GetStaticPaths = async () => {
  const examples = await getExamples();

  const paths = examples.map((example) => ({
    params: { example: example.id },
  }));

  return { paths, fallback: false };
};

export const getStaticProps: GetStaticProps = async (ctx) => {
  const examples = await getExamples();
  const example = examples.find(({ id }) => id === ctx.params?.example);

  if (!example) {
    throw new Error("Could not find example");
  }

  let readme: string | null = null;
  const readmeNode = example.tree.find(
    ({ path, type }) => path === "README.md" && type === "blob"
  );

  if (readmeNode?.url) {
    const readmeRaw = await reqWithSchema(readmeNode.url, blobSchema);

    readme =
      (readmeRaw.encoding === "base64"
        ? Buffer.from(readmeRaw.content, "base64").toString()
        : readmeRaw.content
      ).trim() || null;

    if (readme) {
      // Set a base URL for if using relative links in the README.md file
      const assetsBaseUrl =
        "https://raw.githubusercontent.com/inngest/inngest/main";
      const uiBaseUrl = `https://github.com/inngest/inngest/blob/main/examples/${ctx.params.example}/`;

      readme = marked.parse(
        readme
          /**
           * Replace any image links to `.github/assets/` with an absolute link.
           */
          .replaceAll(".github/assets/", `${assetsBaseUrl}/.github/assets/`)

          /**
           * Remove the very first heading and any whitespace; we want this on
           * GitHub to introduce the quick-start, but here we have our own
           * header for that.
           */
          .replace(/(^#.*\n+)/, ""),
        {
          baseUrl: uiBaseUrl,
          gfm: true,
          breaks: true,
          headerIds: true,
          renderer,
          highlight(code, lang) {
            /**
             * Try get the right highlighting language.
             *
             * GitHub provides `.jsonc` support for pretty comment styling.
             * Highlight.js doesn't, but regular `.json` styling is acceptable,
             * so use that here if we need to.
             */
            const language = hljs.getLanguage(lang)
              ? lang
              : lang.toLowerCase() === "jsonc"
              ? "json"
              : "plaintext";

            return hljs.highlight(code, { language }).value;
          },
        }
      );
    }
  }

  return {
    props: {
      ...example,
      examples,
      readme,
      meta: {
        title: `Quickstart: ${example.name}`,
        description:
          example.description ||
          `Get started using Inngest immediately with the ${example.name} quickstart.`,
      },
    },
  };
};

const Highlights = styled.div`
  background: var(--bg-color-d);
  color: #fff;
`;

const renderer = new marked.Renderer();
const defaultCodeRenderer = renderer.code.bind(renderer);
renderer.code = function (code, language) {
  if (language !== "mermaid") {
    return defaultCodeRenderer(code, language);
  }

  return `<div class="mermaid">${code}</div>`;
};
