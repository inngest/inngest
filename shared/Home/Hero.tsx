import Link from "next/link";
import { ChevronRightIcon } from "@heroicons/react/20/solid";
import clsx from "clsx";
import { CheckIcon } from "@heroicons/react/20/solid";

import Container from "../layout/Container";

/**
 * NOTE - When you update hero copy also update index.tsx's getStaticProps title/description for social & SEO
 */
export default function Hero() {
  return (
    <Container className="mt-12">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row justify-between gap-16 md:gap-24">
        <div className="max-w-[580px] mt-12 mb-12 md:mt-24">
          <h1
            className="pb-8 tracking-tight font-semibold text-4xl md:text-5xl bg-gradient-to-br from-white to-slate-300 bg-clip-text text-transparent"
            style={
              {
                WebkitTextStroke: "0.4px #ffffff80",
                WebkitTextFillColor: "transparent",
                textShadow:
                  "-1px -1px 0 hsla(0,0%,100%,.2), 1px 1px 0 rgba(0,0,0,.1)",
              } as any
            } // silence the experimental webkit props
          >
            {/* Build reliable products */}
            Effortless serverless queues, background jobs, and workflows
          </h1>
          <div className="flex flex-col gap-6 font-normal text-base md:text-lg">
            <p>
              Easily develop serverless workflows in your current codebase,
              without any new infrastructure.
            </p>
            <ul className="flex flex-col gap-2">
              {[
                "Run on serverless, servers or edge",
                "Zero-infrastructure to manage",
                "Automatic retries for max reliability",
              ].map((r) => (
                <li className="flex items-center gap-2" key={r}>
                  <CheckIcon className="h-5 w-5 text-slate-400/80 shrink-0" />{" "}
                  {r}
                </li>
              ))}
            </ul>
            <p>
              Inngest's{" "}
              <Link
                href="/blog/how-durable-workflow-engines-work?ref=homepage-hero"
                className="transition text-indigo-200 hover:text-indigo-300 underline underline-offset-2 decoration-dotted decoration-slate-50/50"
              >
                durable workflow platform
              </Link>{" "}
              and SDKs enable your entire team to ship reliable products.
            </p>
            <div className="flex flex-wrap gap-4 pt-8 text-base">
              <div>
                <Link
                  href="/docs?ref=homepage-hero"
                  className="group rounded-md font-medium px-6 py-2 bg-indigo-500 hover:bg-indigo-400 transition-all text-white whitespace-nowrap flex flex-row items-center"
                >
                  Quick Start Guide{" "}
                  <ChevronRightIcon className="h-5 group-hover:translate-x-1 relative top-px transition-transform duration-150" />
                </Link>
              </div>
              <Link
                href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=homepage-hero`}
                className="rounded-md font-medium px-6 py-2 bg-transparent transition-all text-white border border-slate-800 hover:border-slate-600 hover:bg-slate-500/10 whitespace-nowrap"
              >
                Start Building For Free
              </Link>
            </div>
          </div>
        </div>
        <div className="flex items-center justify-items-center tracking-tight bg-[url(/assets/homepage/hero-paths-graphic.svg)] bg-center	bg-contain bg-no-repeat">
          <div className="lg:min-w-[460px] m-auto grid lg:grid-cols-2 backdrop-blur-sm border border-slate-100/10 border-collapse rounded-lg overflow-hidden font-medium text-md">
            {[
              "Serverless queues",
              "Background jobs",
              "Durable workflows",
              "AI & LLM chaining",
              "Custom workflow engines",
              "Webhook event processing",
            ].map((t, idx, a) => (
              <div
                className={clsx(
                  "min-w-[220px] py-3 px-3 border border-slate-100/10 whitespace-nowrap shadow-lg",
                  idx === 0 && "rounded-t-md lg:rounded-tr-none",
                  idx === 1 && "lg:rounded-tr-md",
                  idx === a.length - 2 && "lg:rounded-bl-md",
                  idx === a.length - 1 && "rounded-b-md lg:rounded-bl-none"
                )}
              >
                {t}
              </div>
            ))}
          </div>
        </div>
      </div>
    </Container>
  );
}
