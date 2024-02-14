import { useRouter } from "next/router";
import shiki from "shiki";
import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";

import Header from "../shared/Header";
import Container from "src/shared/layout/Container";
import PageHeader from "src/shared/PageHeader";
import Footer from "../shared/Footer";

const REPLACE_PATHNAME = "%%PATHNAME%%";

export async function getStaticProps() {
  // Matches mdx/rehyppe.mjs
  const highlighter = await shiki.getHighlighter({ theme: "css-variables" });
  // This can fail locally
  let shikiCodeHtml = "";
  try {
    const shikiTokens = await highlighter.codeToThemedTokens(
      getCode({ pathname: REPLACE_PATHNAME }),
      "typescript"
    );
    shikiCodeHtml = shiki.renderToHtml(shikiTokens, {
      elements: {
        pre: ({ children }) => children,
        code: ({ children }) => children,
        line: ({ children }) => `<span>${children}</span>`,
      },
    });
  } catch (e) {
    // If it fails, just leave it blank
  }

  return {
    props: {
      designVersion: "2",
      shikiCodeHtml,
      meta: {
        title: "404 - Page Not Found",
        description: "",
      },
    },
  };
}

const getCode = ({ pathname }) => {
  return `await inngest.send({
  name: "website/page.not.found",
  data: {
    path: "${pathname}"
  },
  v: "2022-01-16.2",
  ts: ${new Date().valueOf()},
})`;
};

export default function Custom404({ shikiCodeHtml }) {
  const router = useRouter();
  const pathname = router.asPath;
  const isDocs = pathname.match(/^\/docs/);

  // For debugging
  // console.log("404 - Page not found: ", pathname);

  const title = `404 - Page not found`;
  const lede = `We've triggered an event and a serverless function is forwarding it to the team as you read this.`;

  if (isDocs) {
    const html = shikiCodeHtml.replace(REPLACE_PATHNAME, pathname);
    return (
      <>
        <h1>{title}</h1>
        <p>{lede}</p>
        <div className="not-prose my-6 overflow-hidden rounded-lg bg-slate-900 shadow-md">
          <div className="group dark:bg-white/2.5">
            <div className="relative"></div>
            <pre className="overflow-x-auto px-6 py-5 text-xs text-white leading-relaxed">
              <code
                className="language-typescript"
                dangerouslySetInnerHTML={{ __html: html }}
              ></code>
            </pre>
          </div>
        </div>
        <TrackPageNotFound pathname={pathname} />
      </>
    );
  }
  return (
    <div className="font-sans">
      <Header />

      <Container className="my-48">
        <PageHeader title={title} lede={lede} />
        <div className="mt-8">
          <div className="w-full md:w-[400px] xl:w-[360px] xl:mr-10 bg-slate-800/50 backdrop-blur-md border border-slate-700/30 rounded-lg overflow-hidden shadow-lg">
            <div className="flex bg-slate-800/50 items-stretch justify-start gap-2 px-2">
              <div className="border-indigo-400 text-white font-medium text-center text-xs py-2.5 px-2 border-b-[2px]">
                Send 404 Event
              </div>
            </div>
            <SyntaxHighlighter
              language="typescript"
              showLineNumbers={false}
              style={syntaxThemeDark}
              codeTagProps={{ className: "code-window" }}
              customStyle={{
                width: "100%",
                fontSize: "0.8rem",
                padding: "1.5rem",
                backgroundColor: "#0C1323",
                display: "inline-flex",
              }}
            >
              {getCode({ pathname })}
            </SyntaxHighlighter>
          </div>
        </div>
      </Container>
      <TrackPageNotFound pathname={pathname} />
      <Footer />
    </div>
  );
}

const TrackPageNotFound = ({ pathname }) => {
  return (
    <script
      type="text/javascript"
      dangerouslySetInnerHTML={{
        __html: `
        Inngest.event({
          name: "website/page.not.found",
          data: {
            path: "${pathname}",
          },
          v: "2022-01-16.2"
        });
      `,
      }}
    ></script>
  );
};
