import React, { useState } from "react";

import Footer from "src/shared/footer";
import Nav from "src/shared/nav";
import Button from "src/shared/Button";
import CodeWindow from "src/shared/CodeWindow";
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

export const BETA_TYPEFORM_URL = "https://8qe8m10yfz6.typeform.com/to/F1aj8vLl";

export const codesnippets = {
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

export const worksWithBrands = [
  {
    docs: "/docs/guide/nextjs", // TODO - Update final guide url
    logo: "/assets/brand-logos/next-js-dark.svg",
    brand: "Next.js",
    height: "100%",

    type: "framework",
  },
  {
    docs: "/docs/deploying/vercel",
    logo: "/assets/brand-logos/vercel-dark.svg",
    brand: "Vercel",
    height: "50%",
    type: "platform",
  },
  {
    docs: "/docs/deploying/netlify",
    logo: "/assets/brand-logos/netlify-dark.svg",
    brand: "Netlify",
    height: "75%",
    type: "platform",
  },
  {
    docs: "/docs/guide/express",
    logo: "/assets/brand-logos/express-js-dark.svg",
    brand: "Express.js",
    height: "100%",
    type: "framework",
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
        cta={{
          href: "/docs/functions?ref=features-sdk",
          text: "Try the new SDK →",
        }}
        secondaryCTA={{
          href: BETA_TYPEFORM_URL,
          text: "Join the mailing list",
        }}
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
            <CodeWindow
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
                <a href="/docs/functions?ref=features-sdk">Learn more →</a>
              </p>
            </div>

            <CodeWindow
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
                <a href="/docs/deploy?ref=features-sdk">Get more info →</a>
              </p>
            </div>

            <CodeWindow
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
                <a href="/docs/events?ref=features-sdk">Read the docs →</a>
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Background styles */}
      <div className="">
        {/* Content layout */}
        <div className="mx-auto my-28 px-10 lg:px-4 max-w-4xl">
          <header className="mt-24 mb-12 text-center">
            <h2 className="text-4xl">
              Try the{" "}
              <span className="gradient-text gradient-text-ltr">
                Inngest SDK Beta
              </span>
            </h2>
            <p className="mt-8 mx-auto max-w-xl">
              You can try the SDK today! Dive into our docs to get started
            </p>
            <p className="mx-auto max-w-xl">
              Join our mailing list and we'll email you updates and ways to
              provide feedback. You can also join our Discord community to share
              feedback and have a direct line to shaping the future of the SDK!
            </p>
          </header>
          <div className="my-10 flex flex-col sm:flex-row gap-6 justify-center items-center">
            <Button href="/docs?ref=features-sdk" kind="primary">
              Read the docs
            </Button>
            <Button
              href={BETA_TYPEFORM_URL}
              kind="outline"
              style={{ margin: 0 }}
            >
              Join the SDK Beta Mailing List
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
  secondaryCTA,
  ext,
  language,
  onToggle,
}: {
  cta: { href: string; text: string };
  secondaryCTA: { href: string; text: string };
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
          <h1 className="mt-2 mb-6 text-2xl md:text-5xl leading-tight">
            Ship background jobs in seconds
          </h1>
          <p>
            Create and deploy background jobs or scheduled functions right in
            your existing JavaScript or TypeScript codebase.
          </p>
          <p>Works with:</p>
          <div className="mt-4 flex flex-wrap items-center gap-6">
            {worksWithBrands.map((b) => (
              <div className="h-8">
                <img
                  key={b.brand}
                  src={b.logo}
                  alt={`${b.brand}'s logo`}
                  style={{ height: b.height }}
                />
              </div>
            ))}
          </div>
          <div className="mt-10 flex flex-wrap gap-6 justify-start items-center">
            <Button href={cta.href} kind="primary" size="medium">
              {cta.text}
            </Button>
            <Button
              href={secondaryCTA.href}
              kind="outline"
              size="medium"
              style={{ margin: 0 }}
            >
              {secondaryCTA.text}
            </Button>
          </div>
        </header>
        <div className="lg:mt-12 mx-auto lg:mx-6 max-w-full md:max-w-lg flex flex-col justify-between">
          <CodeWindow
            className="transform-iso shadow-xl relative z-10"
            filename={`myGreatFunction.${ext}`}
            snippet={codesnippets[language].function}
          />
          <CodeWindow
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
