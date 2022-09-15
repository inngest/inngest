import React, { useState } from "react";
import styled from "@emotion/styled";
import SyntaxHighlighter from "react-syntax-highlighter";
import {
  atomOneLight as syntaxThemeLight,
  atomOneDark as syntaxThemeDark,
} from "react-syntax-highlighter/dist/cjs/styles/hljs";

import Footer from "src/shared/footer";
import Nav from "src/shared/nav";
import Button from "src/shared/Button";
import CheckRounded from "src/shared/Icons/CheckRounded";
import Discord from "src/shared/Icons/Discord";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        // TODO - Final title
        title: "Inngest JavaScript & TypeScript SDK",
        description:
          "Quickly build, test and deploy code that runs in response to events or on a schedule — without spending any time on infrastructure.",
        image: "/assets/img/og-image-default.jpg",
      },
    },
  };
}

const BETA_TYPEFORM_URL = "https://8qe8m10yfz6.typeform.com/to/F1aj8vLl";

const codesnippets = {
  javascript: {
    function: `
      import { createFunction } from "inngest"

      const myFn = async ({ event }) => {
        return { message: "success" }
      }

      export default createFunction(
        "My Great Function",
        "demo/event.name",
        myFn
      )
    `,
    sendEventShort: `
      import { Inngest } from "inngest"
      const inngest = new Inngest("sourceKey123")
      inngest.send({
        name: "demo/event.name",
        data: { something: req.body.something }
      })
    `,
    sendEvent: `
      import { Inngest } from "inngest"

      const inngest = new Inngest("<SOURCE_KEY>")

      export default function apiEndpoint(req, res) {
        const success = yourExistingCode(req.body)
        inngest.send({
          name: "demo/event.name",
          data: { something: req.body.something }
        })
        res.status(200).json({ success })
      }
    `,
    nextJSHandler: `
      import { Inngest } from "inngest"

      import { myGreatFunction } from "../../myGreatFunction"
      import { scheduledTask } from "../../scheduledTask"

      const inngest = new Inngest("My App", "<API_KEY>")

      export default register(inngest, "<SIGNING_KEY>", [
        myGreatFunction,
        scheduledFunction
      ]);
    `,
  },
  typescript: {
    function: `
      import { createFunction } from "inngest"
      import { DemoEventTrigger } from "./types"

      const myFn = async ({ event }) => {
        return { message: "success" }
      }

      export default createFunction<DemoEventTrigger>(
        "My Great Function",
        "demo/event.trigger",
        myFn
      )
    `,
    sendEventShort: `
      import { Inngest } from "inngest"
      import { Events } from "../../__generated__/inngest"
      const inngest = new Inngest<Events>("sourceKey123")
      inngest.send({
        name: "demo/event.name",
        data: { something: req.body.something }
      })
    `,
    sendEvent: `
      import { Inngest } from "inngest"
      import { Events } from "../../__generated__/inngest"

      const inngest = new Inngest<Events>("sourceKey123")

      inngest.send({
        name: "demo/event.name",
        data: { something: req.body.something }
      })
    `,
    nextJSHandler: `
      import { Inngest } from "inngest"
      import { Events } from "../../__generated__/inngest"

      import { myGreatFunction } from "../../myGreatFunction"
      import { scheduledTask } from "../../scheduledTask"

      const inngest = new Inngest<Events>("My App", "<API_KEY>")

      export default register(inngest, "<SIGNING_KEY>", [
        myGreatFunction,
        scheduledFunction
      ]);
    `,
  },
};

const removeLeadingSpaces = (snippet: string): string => {
  const lines = snippet.split("\n");
  if (!lines[0].replace(/^\s+/, "").length) {
    lines.shift();
  }
  if (!lines[lines.length - 1].replace(/^\s+/, "").length) {
    lines.pop();
  }
  const leadingSpace = lines[0].match(/^\s+/)?.[0];
  return lines.map((l) => l.replace(leadingSpace, "")).join("\n");
};

const worksWithBrands = [
  {
    logo: "/assets/brand-logos/next-js-dark.svg",
    brand: "Next.js",
    className: "h-8",
  },
  {
    logo: "/assets/brand-logos/vercel-dark.svg",
    brand: "Vercel",
    className: "h-4",
  },
  {
    logo: "/assets/brand-logos/netlify-dark.svg",
    brand: "Netlify",
    className: "h-6",
  },
  {
    logo: "/assets/brand-logos/express-js-dark.svg",
    brand: "Express.js",
    className: "h-8",
  },
];

export default function FeaturesSDK() {
  const [language, setLanguage] = useState<"javascript" | "typescript">(
    "javascript"
  );
  const ext = language === "typescript" ? "ts" : "js";
  return (
    <div>
      <Nav sticky={true} />

      <Hero
        cta={{ href: BETA_TYPEFORM_URL, text: "Join the beta →" }}
        language={language}
        ext={ext}
        onToggle={(l) => setLanguage(l)}
      />

      {/* Background styles */}
      <div className="bg-light-gray background-grid-texture">
        {/* Content layout */}
        <div className="mx-auto my-14 py-24 px-10 lg:px-4 max-w-4xl">
          <header className="mb-12 text-center">
            <h2 className="text-4xl">Simple, but powerful</h2>
          </header>

          <div className="mb-12 text-center">
            <LanguageToggle
              onClick={(l) => setLanguage(l)}
              language={language}
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 md:gap-y-12">
            <CodeBlock
              className="shadow-xl hover:transform-iso-opposite"
              filename={`inngest/myGreatFunction.${ext}`}
              snippet={codesnippets[language].function}
            />
            <div>
              <h3 className="text-2xl">Run your code in response to events</h3>
              <p className="my-6">
                Easily define functions that run in response to events.
                Background jobs, scheduled tasks, webhooks are all made easy
                with Inngest's SDK.
              </p>
              <p>
                <a href={BETA_TYPEFORM_URL}>Learn more →</a>
              </p>
            </div>

            <CodeBlock
              className="shadow-xl hover:transform-iso-opposite"
              filename={`api/inngest.${ext}`}
              snippet={codesnippets[language].nextJSHandler}
            />
            <div>
              <h3 className="text-2xl">Deploy to your existing setup</h3>
              <p className="my-6">
                Inngest can remotely run your background tasks via secure HTTP
                handler. You keep your existing deployment workflow and we'll
                call your code where it is.
              </p>
              <p>
                <a href={BETA_TYPEFORM_URL}>Get more info →</a>
              </p>
            </div>

            <CodeBlock
              className="shadow-xl hover:transform-iso-opposite"
              filename={`api/someEndpoint.${ext}`}
              snippet={codesnippets[language].sendEvent}
            />
            <div>
              <h3 className="text-2xl">
                Trigger jobs with events from anywhere
              </h3>
              <p className="my-6">
                Send events with our SDK to trigger background jobs and move
                longer running code out of the critical path of an API request.
              </p>
              <p>
                <a href={BETA_TYPEFORM_URL}>Join the beta user list →</a>
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Background styles */}
      <div className="">
        {/* Content layout */}
        <div className="mx-auto my-28 px-10 lg:px-4 max-w-4xl">
          <header className="my-24 text-center">
            <h2 className="text-4xl">
              The Complete Platform For{" "}
              <span className="gradient-text gradient-text-ltr gradient-from-pink gradient-to-orange">
                Everything&nbsp;Async
              </span>
            </h2>
            <p className="mt-8">
              Our serverless solution provides everything you need to
              effortlessly
              <br />
              build and manage every type of asynchronous and event-driven job.
            </p>
          </header>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 lg: gap-y-12">
            <div className="md:h-48 flex flex-col justify-center items-center">
              <div
                className="w-72 relative grid grid-cols-8 gap-0 transform-iso-opposite rounded-lg border-4 border-transparent"
                style={{
                  maxWidth: "340px",
                  background:
                    "linear-gradient(#fff, #fff) padding-box, linear-gradient(to right, #5D5FEF, #EF5F5D) border-box",
                }}
              >
                <div className="absolute right-1" style={{ top: "-3rem" }}>
                  <CheckRounded fill="#5D5FEF" size="5rem" />
                </div>
                {[1, 2, 3, 4, 5, 6, 7, 8].map((n) => (
                  <div
                    key={n}
                    className={`h-12 bg-white border-slate-200 ${
                      n !== 8 ? "border-r-4" : "rounded-r-md"
                    } ${n === 1 ? "rounded-l-md" : ""}
                    `}
                    style={{
                      animation: `queue-message-flash 4s infinite ${n / 2}s`,
                    }}
                  >
                    &nbsp;
                  </div>
                ))}
              </div>
            </div>
            <div>
              <h3 className="text-2xl">No infrastructure to manage</h3>
              <p className="my-6">
                Inngest is serverless, requiring absolutely no infra for you to
                manage. No queues, event bus, or logging to configure.
              </p>
            </div>

            <div className="md:h-48 flex flex-col justify-center items-center">
              <img
                src="/assets/homepage/admin-ui-screenshot.png"
                className="rounded-sm transform-iso-opposite"
                style={{ maxWidth: "340px" }}
              />
            </div>
            <div>
              <h3 className="text-2xl">
                A real-time dashboard keeps everyone in the loop
              </h3>
              <p className="my-6">
                The Inngest Cloud Dashboard brings full transparency to all your
                asynchronous jobs, so you can stay on top of performance,
                throughput, and more, without needing to dig through logs.
              </p>
            </div>

            <div className="md:h-48 flex flex-col justify-center items-center">
              <div
                className="transform-iso-opposite flex flex-col gap-1"
                style={{ boxShadow: "none" }}
              >
                {[
                  "Automatic Retries",
                  "Event Replay",
                  "Versioning",
                  "Idempotency",
                ].map((s) => (
                  <div key={s} className="flex flex-row items-center gap-2">
                    <CheckRounded fill="#5D5FEF" size="1.6rem" shadow={false} />{" "}
                    {s}
                  </div>
                ))}
              </div>
            </div>
            <div>
              <h3 className="text-2xl">
                Event-driven, as easy as just sending events!
              </h3>
              <p className="my-6">
                We built all the hard stuff so you don't have to: idempotency,
                throttling, backoff, retries,{" "}
                <a href="/blog/introducing-cli-replays?ref=homepage">replays</a>
                , job versioning, and so much more. With Inngest, you just write
                your code and we take care of the rest.
              </p>
            </div>
          </div>
          <div className="my-10 flex justify-center">
            <Button href={BETA_TYPEFORM_URL} kind="outlinePrimary">
              Join the SDK Beta →
            </Button>
          </div>
        </div>
      </div>

      {/* Background styles */}
      <div className="">
        {/* Content layout */}
        <div className="mx-auto my-28 px-10 lg:px-4 max-w-4xl">
          <header className="mt-24 mb-12 text-center">
            <h2 className="text-4xl">
              Join the{" "}
              <span className="gradient-text gradient-text-ltr">
                Inngest SDK Beta
              </span>
            </h2>
            <p className="mt-8 mx-auto max-w-xl">
              We'll be releasing the SDK beta for JavaScript and TypeScript
              throughout September.
            </p>
            <p className="mx-auto max-w-xl">
              Join our list and we'll email you when it's ready to test and
              provide feedback on. You can also join our Discord community to
              share feedback and have a direct line to shaping the future of the
              SDK!
            </p>
          </header>
          <div className="my-10 flex flex-col sm:flex-row gap-6 justify-center items-center">
            <Button href={BETA_TYPEFORM_URL} kind="primary">
              Join the SDK Beta →
            </Button>
            <Button
              href="https://www.inngest.com/discord"
              kind="outline"
              style={{ margin: 0 }}
            >
              <Discord /> Join our community on Discord
            </Button>
          </div>
        </div>
      </div>

      <Footer />
    </div>
  );
}

export const Hero = ({
  cta,
  ext,
  language,
  onToggle,
}: {
  cta: { href: string; text: string };
  language: "javascript" | "typescript";
  ext: string;
  onToggle: (string) => void;
}) => {
  return (
    <div>
      {/* Content layout */}
      <div className="mx-auto my-12 px-10 lg:px-16 max-w-5xl grid grid-cols-1 lg:grid-cols-2 gap-8">
        <header className="lg:my-24 mt-8">
          <span className="text-sm font-bold uppercase gradient-text-ltr">
            Inngest SDK Beta
          </span>
          <h1 className="mt-2 mb-6 text-5xl leading-tight">
            Ship background jobs in seconds
          </h1>
          <p>
            Create and deploy background jobs or scheduled functions right in
            your existing JavaScript or TypeScript codebase.
          </p>
          <p>Works with:</p>
          <div className="mt-4 flex items-center gap-6">
            {worksWithBrands.map((b) => (
              <img
                key={b.brand}
                src={b.logo}
                alt={`${b.brand}'s logo`}
                className={b.className}
              />
            ))}
          </div>
          <div className="mt-10 flex h-10">
            <Button href={cta.href} kind="primary" size="medium">
              {cta.text}
            </Button>
            {/*<Button href="/docs/sdk" kind="outline" size="medium">
                Read the docs
              </Button>*/}
          </div>
        </header>
        <div className="lg:mt-12 mx-auto lg:mx-6 max-w-lg flex flex-col justify-between">
          <CodeBlock
            className="transform-iso shadow-xl relative z-10"
            filename={`myGreatFunction.${ext}`}
            snippet={codesnippets[language].function}
          />
          <CodeBlock
            className="mt-6 transform-iso-opposite shadow-xl relative"
            filename={`api/someEndpoint.${ext}`}
            snippet={codesnippets[language].sendEventShort}
          />
          <div className="mt-12 text-center">
            <LanguageToggle onClick={(l) => onToggle(l)} language={language} />
          </div>
        </div>
      </div>
    </div>
  );
};

const LanguageToggle = ({ onClick, language = "javascript" }) => {
  const options = [
    { key: "javascript", name: "JavaScript" },
    { key: "typescript", name: "TypeScript" },
  ];
  return (
    <div className="inline-flex text-xs rounded-md overflow-hidden border-solid border-2 border-indigo-600">
      {options.map((o) => (
        <button
          key={o.key}
          className={`py-1 px-2 ${language === o.key ? "bg-violet-200" : ""}`}
          onClick={() => onClick(o.key)}
        >
          {o.name}
        </button>
      ))}
    </div>
  );
};

const CodeBlock = ({
  snippet,
  className = "",
  filename = "",
  theme = "light",
}: {
  snippet: string;
  className?: string;
  filename?: string;
  theme?: "light" | "dark";
}) => {
  const backgroundColor =
    theme === "dark"
      ? "var(--color-almost-black)"
      : "var(--color-almost-white)";
  return (
    <div
      className={`p-2 ${className}`}
      style={{ backgroundColor, borderRadius: "var(--border-radius)" }}
    >
      <div className="mb-1 flex gap-1 relative">
        <div className="w-2.5 h-2.5 border border-slate-700 rounded-full"></div>
        <div className="w-2.5 h-2.5 border border-slate-700 rounded-full"></div>
        <div className="w-2.5 h-2.5 border border-slate-700 rounded-full"></div>
        <div
          className="text-slate-500 absolute inset-x-0 mx-auto text-center"
          style={{ fontSize: "0.6rem", top: "-1px" }}
        >
          {filename}
        </div>
      </div>
      <SyntaxHighlighter
        language="javascript"
        showLineNumbers={true}
        style={theme === "dark" ? syntaxThemeDark : syntaxThemeLight}
        customStyle={{
          backgroundColor,
          fontSize: "0.7rem",
        }}
      >
        {removeLeadingSpaces(snippet)}
      </SyntaxHighlighter>
    </div>
  );
};
