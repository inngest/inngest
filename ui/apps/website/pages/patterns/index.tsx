import React from "react";
import styled from "@emotion/styled";
import Link from "next/link";

import Header from "../../shared/Header";
import Footer from "../../shared/Footer";
import Container from "../../shared/layout/Container";
import ArrowRight from "src/shared/Icons/ArrowRight";

interface Section {
  title: string;
  articles: {
    title: string;
    subtitle: string;
    tags: string[];
    slug: string;
  }[];
}

export const SECTIONS: Section[] = [
  {
    title: "The Basics",
    articles: [
      {
        title: "Keeping your API fast",
        subtitle:
          "Moving code out of the critical path into background jobs to keep API response times performant",
        tags: ["Performance"],
        slug: "keeping-your-api-fast",
      },
      {
        title: "Running code on a schedule",
        subtitle:
          "Run task periodically, as cron jobs, like weekly emails or daily backups",
        tags: ["Scheduling"],
        slug: "running-code-on-a-schedule",
      },
      {
        title: "Build reliable webhooks",
        subtitle: "Handle high throughput webhooks in a fault tolerant way",
        tags: ["Performance", "Reliability"],
        slug: "build-reliable-webhooks",
      },
      {
        title: "Reliably run critical workflows",
        subtitle: "Break complex code into reliable, independently ran steps",
        tags: ["Reliability"],
        slug: "reliably-run-critical-workflows",
      },
    ],
  },
  {
    title: "The Advantage of Events",
    articles: [
      {
        title: "Running functions in parallel",
        subtitle: "Trigger multiple functions from a single event",
        tags: ["Architecture"],
        slug: "running-functions-in-parallel",
      },
      {
        title: "Running at specific times",
        subtitle:
          "Pause and wait until a specific time based off of data within an event",
        tags: ["Scheduling", "Architecture"],
        slug: "running-at-specific-times",
      },
      {
        title: "Cancelling scheduled functions",
        subtitle:
          "Automatically cancel scheduled, paused, and waiting work using events",
        tags: ["Scheduling"],
        slug: "cancelling-scheduled-functions",
      },
      /*{
        title: "Data recovery through replay",
        subtitle: "Use event history to re-run work to fix issues",
        tags: ["Architecture"],
        slug: "#TODO",
      },*/
      {
        title: "Batching jobs via fan-out",
        subtitle:
          "Reliably manage thousands of jobs triggered by a single event or cron",
        tags: ["Architecture", "Scheduling"],
        slug: "reliable-scheduling-systems",
      },
    ],
  },
  {
    title: "Event-coordination",
    articles: [
      {
        title: "Building flows for lost customers",
        subtitle:
          "Combine events into a single function to build things like cart abandonment, sales processes, and churn flows",
        tags: ["Activation", "User Journeys", "Event Coordination"],
        slug: "event-coordination-for-lost-customers",
      },
      /*{
        title: "Human-in-the-middle",
        subtitle: "Workflows that require human input to run conditional code",
        tags: ["Compliance", "Internal Tooling"],
        slug: "#TODO",
      },*/
    ],
  },
];

const zeroPad = (n: number, digits = 2): string => {
  const ns = n.toString();
  const len = ns.length;
  return len >= digits ? ns : `${new Array(digits - len + 1).join("0")}${n}`;
};

export async function getStaticProps() {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Patterns: Async + Event-Driven",
        description:
          "A collection of software architecture patterns for asynchronous flows",
        image: "/assets/patterns/og-image-patterns.jpg",
      },
    },
  };
}

export default function Patterns() {
  return (
    <div>
      <Header />

      <div
        style={{
          backgroundImage: "url(/assets/pricing/table-bg.png)",
          backgroundPosition: "center -30px",
          backgroundRepeat: "no-repeat",
          backgroundSize: "1800px 1200px",
        }}
      >
        <Container>
          <h1 className="text-3xl lg:text-5xl text-white mt-12 md:mt-20 font-semibold tracking-tight">
            Patterns
          </h1>
          <p className="text-xl text-slate-100">Async + Event-Driven</p>
          <p className="my-4 text-indigo-200 max-w-xl  mb-16 md:mb-28">
            The common patterns listed here are flexible and powerful enough to
            solve problems across all types of projects and codebases.
          </p>

          <section className="flex flex-col gap-12">
            {/* Content layout */}

            {SECTIONS.map((s, idx) => (
              <div
                key={s.title}
                className="gap-y-6 xl:gap-y-8 flex flex-col xl:grid xl:grid-cols-4 md:bg-slate-900/20 rounded-lg md:px-3 md:py-6 lg:p-6"
              >
                <div className="flex items-center gap-4 xl:block">
                  <div className="w-10 h-10 bg-indigo-500 flex items-center justify-center text-lg font-bold text-white rounded">
                    {zeroPad(idx + 1)}
                  </div>
                  <h2 className="text-xl xl:mt-4 text-white font-medium tracking-tight">
                    {s.title}
                  </h2>
                </div>
                <div className="grid col-span-3 md:grid-cols-2 gap-x-6 gap-y-6">
                  {s.articles.map(({ title, subtitle, tags, slug }) => (
                    <Link
                      key={slug}
                      href={`/patterns/${slug}`}
                      className="flex flex-col justify-between bg-slate-900 rounded-lg hover:bg-slate-50 group/card transition-all"
                    >
                      <div className="px-6 py-4 lg:px-8 lg:py-6 h-full flex flex-col justify-between">
                        <div>
                          <h2 className="text-lg text-white group-hover/card:text-slate-700 font-semibold tracking-tight">
                            {title}
                          </h2>
                          <p className="text-sm mt-1 mb-3 text-indigo-200 group-hover/card:text-slate-500 font-regular tracking-tight">
                            {subtitle}.
                          </p>
                        </div>
                        <span className="text-sm text-indigo-400 font-medium group-hover/card:text-indigo-500 transition-all flex items-center gap-1">
                          Read pattern
                          <ArrowRight className="group-hover/card:translate-x-1 transition-transform duration-150  -mr-1.5" />
                        </span>
                      </div>
                      <div className="flex flex-wrap gap-2 bg-slate-950 group-hover/card:bg-slate-100  rounded-b-lg py-3 px-6 border-t border-slate-800/60 group-hover/card:border-slate-200 transition-all">
                        {tags.map((t) => (
                          <span
                            key={t}
                            className="py-1 px-2 rounded bg-slate-800 text-slate-300 group-hover/card:bg-slate-200 group-hover/card:text-slate-500 transition-all font-medium text-xs"
                          >
                            {t}
                          </span>
                        ))}
                      </div>
                    </Link>
                    // <Link
                    //   key={slug}
                    //   href={`/patterns/${slug}`}
                    //   className="flex flex-col bg-white rounded-lg"
                    // >
                    //   <div className="px-8 py-6">
                    //     <h2 className="text-lg text-slate-700 font-semibold tracking-tight">
                    //       {title}
                    //     </h2>
                    //     <p className="text-sm mt-1 mb-3 text-slate-600 font-regular tracking-tight">
                    //       {subtitle}.
                    //     </p>
                    //     <span className="text-sm text-indigo-500 font-medium">
                    //       Read pattern
                    //     </span>
                    //   </div>
                    //   <div className="flex gap-2 bg-slate-100 rounded-b-lg py-3 px-6 border-t border-slate-200/60">
                    //     {tags.map((t) => (
                    //       <span
                    //         key={t}
                    //         className="py-1 px-2 rounded bg-slate-200 text-slate-600 font-medium text-xs"
                    //       >
                    //         {t}
                    //       </span>
                    //     ))}
                    //   </div>
                    // </Link>
                  ))}
                </div>
              </div>
            ))}
          </section>
        </Container>
      </div>
      <Footer />
    </div>
  );
}
