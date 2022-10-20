import { useState } from "react";

import Button from "src/shared/Button";
import CodeWindow from "src/shared/CodeWindow";
import Footer from "src/shared/footer";
import CheckboxUnchecked from "src/shared/Icons/CheckboxUnchecked";
import CheckRounded from "src/shared/Icons/CheckRounded";
import Discord from "src/shared/Icons/Discord";
import Nav from "src/shared/nav";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        // TODO - Final title
        title: "Inngest JavaScript & TypeScript SDK",
        description:
          "Quickly build, test and deploy code that runs in response to events or on a schedule — without spending any time on infrastructure.",
      },
    },
  };
}

export const BETA_TYPEFORM_URL = "https://8qe8m10yfz6.typeform.com/to/F1aj8vLl";

export const codesnippets = {
  javascript: {
    function: `
      import { createFunction } from "inngest"

      export default createFunction(
        "My Great Function",
        "demo/event.name",
        async ({ event }) => {
          return { message: "success" }
        }
      )
    `,
    scheduledFunction: `
      import { createScheduledFunction } from "inngest"

      const scheduledTask = () => {
        return { message: "success" }
      }

      export default createScheduledFunction(
        "A scheduled function",
        "0 9 * * MON", // any cron expression
        scheduledTask
      )
    `,
    sendEventShort: `
      import { Inngest } from "inngest"
      const inngest = new Inngest({ name: "My App" })
      inngest.send({
        name: "demo/event.name",
        data: { something: req.body.something }
      })
    `,
    sendEvent: `
      import { Inngest } from "inngest"

      const inngest = new Inngest({ name: "My App" })

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
      import { serve } from "inngest/next"

      import { myGreatFunction } from "../../myGreatFunction"
      import { scheduledTask } from "../../scheduledTask"

      export default serve("My App", [
        myGreatFunction,
        scheduledTask
      ])
    `,
  },
  typescript: {
    function: `
      import { createFunction } from "inngest"
      import { DemoEventTrigger } from "./types"

      export default createFunction<DemoEventTrigger>(
        "My Great Function",
        "demo/event.trigger",
        async ({ event }) => {
          return { message: "success" }
        }
      )
    `,
    scheduledFunction: `
      import { createScheduledFunction } from "inngest"

      const scheduledTask = () => {
        return { message: "success" }
      }

      export default createScheduledFunction(
        "A scheduled function",
        "0 9 * * MON", // any cron expression
        scheduledTask
      )
    `,
    sendEventShort: `
      import { Inngest } from "inngest"
      import { Events } from "../../__generated__/inngest"
      const inngest = new Inngest<Events>({ name: "My App" })
      inngest.send({
        name: "demo/event.name",
        data: { something: req.body.something }
      })
    `,
    sendEvent: `
      import { Inngest } from "inngest"
      import { Events } from "../../__generated__/inngest"

      const inngest = new Inngest<Events>({ name: "My App" })

      inngest.send({
        name: "demo/event.name",
        data: { something: req.body.something }
      })
    `,
    nextJSHandler: `
      import { Inngest } from "inngest"
      import { serve } from "inngest/next"
      import { Events } from "../../__generated__/inngest"

      import { myGreatFunction } from "../../myGreatFunction"
      import { scheduledTask } from "../../scheduledTask"

      const inngest = new Inngest<Events>({ name: "My App" })

      export default serve(inngest, [
        myGreatFunction,
        scheduledTask
      ])
    `,
  },
};

export const worksWithBrands = [
  {
    docs: "/docs/frameworks/nextjs",
    logo: "/assets/brand-logos/next-js-dark.svg",
    brand: "Next.js",
    height: "50%",
    type: "framework",
  },
  {
    docs: "/docs/deploy",
    logo: "/assets/brand-logos/vercel-dark.svg",
    brand: "Vercel",
    height: "50%",
    type: "platform",
  },
  {
    docs: "/docs/deploy/netlify",
    logo: "/assets/brand-logos/netlify-dark.svg",
    brand: "Netlify",
    height: "65%",
    type: "platform",
  },
  {
    docs: "/docs/frameworks/express", // TODO - change to guide when launched
    logo: "/assets/brand-logos/express-js-dark.svg",
    brand: "Express.js",
    height: "80%",
    type: "framework",
  },
  {
    docs: "/docs/frameworks/cloudflare-pages",
    logo: "/assets/brand-logos/cloudflare-dark.svg",
    brand: "Cloudflare Pages",
    height: "80%",
    type: "platform",
  },
];

export default function FeaturesSDK() {
  const [language, setLanguage] = useState<"javascript" | "typescript">(
    "typescript"
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
        <div className="mx-auto mt-14 py-24 px-10 lg:px-4 max-w-4xl">
          <header className="mb-12 text-center">
            <h2 className="text-4xl">Simple, yet powerful</h2>
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

            <CodeWindow
              className="shadow-xl hover:transform-iso-opposite"
              filename={`inngest/scheduledTask.${ext}`}
              snippet={codesnippets[language].scheduledFunction}
            />
            <div>
              <h3 className="text-2xl">Schedule jobs easily</h3>
              <p className="my-6">
                Run your code on a schedule using basic cron expressions. Cron
                jobs made simple with observability & logs.
              </p>
              <p>
                <a href="/docs/functions?ref=features-sdk">Learn how →</a>
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
          </div>
        </div>
      </div>

      {/* Background styles */}
      <div className="bg-[#070707] background-grid-texture text-white">
        {/* Content layout */}
        <div className="mx-auto mb-14 py-24 px-10 lg:px-4 max-w-4xl">
          <header className="mb-12 text-center">
            <h2 className="text-4xl">
              Deep{" "}
              <span className="gradient-text gradient-text-ltr gradient-from-blue gradient-to-cyan">
                TypeScript
              </span>{" "}
              support
            </h2>
          </header>

          <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 lg:gap-8 items-center">
            <div className="lg:col-span-5 grid grid-cols-1 gap-8 justify-center">
              {[
                {
                  title: (
                    <img
                      className="w-6 h-6 inline mr-1"
                      src="/assets/brand-logos/typescript.svg"
                    />
                  ),
                  subtitle: "Painless type generation",
                  description:
                    "One command generates TS types for your entire ecosystem, based on your actual production data.",
                },
                {
                  title: "🛡️",
                  subtitle: "Automatic typesafety",
                  description:
                    "Keep in sync with your production data; strict TS types guard you every step of the way.",
                },

                {
                  title: "🧙",
                  subtitle: "Smart autocomplete",
                  description: (
                    <>
                      Leverage TypeScript inference and easily discover events
                      and their data using autocomplete.
                    </>
                  ),
                },
              ].map((i) => (
                <div className="w-full flex flex-col space-y-4">
                  <h3 className="text-xl font-bold text-white">
                    <span className="text-2xl mr-2">{i.title}</span>
                    {i.subtitle}
                  </h3>
                  <div className="text-sm">{i.description}</div>
                </div>
              ))}
            </div>

            <div className="lg:col-span-7 flex items-center justify-center">
              <img
                src="/assets/next-test.gif"
                className="shadow-xl rounded"
                alt="A screencast of the Inngest SDK's TypeScript features"
              />
            </div>
          </div>

          <div className="flex items-center justify-center mt-16">
            <Button kind="primary" href="/docs/typescript">
              Learn more →
            </Button>
          </div>
        </div>
      </div>

      {/* Background styles */}
      <div className="">
        {/* Content layout */}
        <div className="mx-auto my-16 py-10 px-6 md:px-16 max-w-xl rounded-lg bg-gradient-to-br from-pink-50 to-violet-50">
          <h2 className="text-xl sm:text-4xl mt-2 mb-2">Inngest SDK Roadmap</h2>
          <p className="text-sm text-color-secondary">
            What we've built and what's up next:
          </p>
          <ul className="my-4 flex flex-col gap-4">
            {[
              {
                done: true,
                text: "Create event-driven and scheduled functions",
              },
              {
                done: true,
                text: "Send events",
              },
              {
                done: true,
                text: "TypeScript support, including generics",
              },
              {
                done: true,
                text: (
                  <>
                    <a href="/docs/frameworks/nextjs?ref=features-sdk">
                      Next.js
                    </a>{" "}
                    & <a href="/docs/frameworks/express">Express.js</a> support
                  </>
                ),
              },
              {
                done: true,
                text: (
                  <>
                    TypeScript: Event <code>Type</code> generation and sync
                  </>
                ),
                docs: "/docs/typescript",
              },
              {
                done: true,
                text: "Cloudflare Pages support",
                docs: "/docs/deploy/cloudflare",
              },
              { done: true, text: "Inngest local dev server integration" },
              { done: false, text: "Step Functions" },
              {
                done: false,
                text: "Step delays, conditional expressions, & event-coordination",
              },
              { done: false, text: "Inngest Cloud deploy" },
            ].map((i, idx) => (
              <ol
                key={`item-${idx}`}
                className="flex flex-row items-center gap-2"
              >
                {i.done ? (
                  <CheckRounded
                    size="22"
                    fill="#5D5FEF"
                    shadow={false}
                    stroke={10}
                  />
                ) : (
                  <CheckboxUnchecked size="22" fill="#5D5FEF" />
                )}
                <div>
                  {i.text}{" "}
                  {i.docs && (
                    <>
                      (<a href={`${i.docs}?ref=features-sdk-roadmap`}>docs</a>)
                    </>
                  )}
                </div>
              </ol>
            ))}
          </ul>
          <p>
            <em>Future:</em> Additional platform support (AWS Lambda, Supabase,
            Deno), additional framework support (Remix, RedwoodJS), testing APIs
          </p>
        </div>
      </div>

      {/* Background styles */}
      <div className="">
        {/* Content layout */}
        <div className="mx-auto my-28 px-10 lg:px-4 max-w-4xl grid md:grid-cols-3 gap-8">
          {[
            {
              type: "Framework Guide",
              name: "Next.js",
              logo: "/assets/brand-logos/next-js-dark.svg",
              logoHeight: worksWithBrands.find((b) => b.brand === "Next.js")
                .height,
              description:
                "Trigger background and scheduled functions using our Next.js adapter",
              href: "/docs/frameworks/nextjs",
            },
            {
              type: "Framework Guide",
              name: "Express.js",
              logo: "/assets/brand-logos/express-js-dark.svg",
              logoHeight: worksWithBrands.find((b) => b.brand === "Express.js")
                .height,
              description:
                "Run background jobs without workers or setting up a queue",
              href: "/docs/deploy/express",
            },
            {
              type: "Platform Guide",
              name: "Netlify",
              logo: "/assets/brand-logos/netlify-dark.svg",
              logoHeight: worksWithBrands.find((b) => b.brand === "Netlify")
                .height,
              description:
                "Use our Netlify plugin to automate deployments of your functions",
              href: "/docs/deploy/netlify",
            },
          ].map((i) => (
            <a
              href={`${i.href}?ref=features-sdk-guides`}
              className="text-color-primary flex transition-all	hover:-translate-y-1"
            >
              <div className="p-6 h-full w-full flex flex-col rounded-md border-2 border-slate-800">
                <p>
                  <span
                    className={`text-xs font-bold uppercase gradient-text-ltr ${
                      i.type.match(/framework/i)
                        ? "gradient-from-blue gradient-to-cyan"
                        : "gradient-from-pink gradient-to-orange"
                    }`}
                  >
                    {i.type}
                  </span>
                </p>
                <div className="h-10 my-4 flex items-center">
                  <img
                    src={i.logo}
                    style={{ height: i.logoHeight }}
                    alt={i.name}
                  />
                </div>
                <p className="mb-6 text-sm">{i.description}</p>
                <p className="mt-auto">
                  {/* The whole div is clickable, but we make this look like a link to visually show it's clickable */}
                  <span className="text-color-iris-100">Read the guide →</span>
                </p>
              </div>
            </a>
          ))}
        </div>
      </div>

      {/* Background styles */}
      <div className="">
        {/* Content layout */}
        <div className="mx-auto my-28 px-6 lg:px-4 max-w-4xl">
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
          <div className="my-10 flex flex-col sm:flex-row flex-wrap gap-6 justify-center items-center">
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
            Write functions, send events.
          </h1>
          <p>
            Create and deploy background jobs or scheduled functions right in
            your existing JavaScript or TypeScript codebase.
          </p>
          <p>Works with:</p>
          <div className="mt-4 flex flex-wrap items-center gap-6">
            {worksWithBrands.map((b) => (
              <a
                href={`${b.docs}?ref=features-sdk-hero`}
                className="h-7 flex items-center bulge"
              >
                <img
                  key={b.brand}
                  src={b.logo}
                  alt={`${b.brand}'s logo`}
                  style={{ height: b.height }}
                />
              </a>
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
    { key: "typescript", name: "TypeScript" },
    { key: "javascript", name: "JavaScript" },
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
