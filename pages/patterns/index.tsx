import React from "react";
import styled from "@emotion/styled";
import Link from "next/link";

import Nav from "src/shared/nav";
import Footer from "src/shared/Footer";

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
        subtitle: "Fan-out work to multiple functions using a single event",
        tags: ["Architecture"],
        slug: "running-functions-in-parallel",
      },
      {
        title: "Running at specific times",
        subtitle: "Pause and wait until a specific time based off of data within an event",
        tags: ["Scheduling", "Architecture"],
        slug: "running-at-specific-times",
      },
      {
        title: "Cancelling scheduled functions",
        subtitle: "Automatically cancel scheduled, paused, and waiting work using events",
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
        title: "Reliable scheduling systems",
        subtitle:
          "Combine cron-jobs with event fan-out for auditable scheduling",
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
      {
        title: "Human-in-the-middle",
        subtitle: "Workflows that require human input to run conditional code",
        tags: ["Compliance", "Internal Tooling"],
        slug: "#TODO",
      },
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
  const sectionClasses = `max-w-2xl mx-auto text-left`;
  return (
    <Page>
      <Nav sticky={true} />

      <Content className="py-28">
        {/* Background styles */}
        <div>
          {/* Content layout */}
          <div className={`${sectionClasses}mb-14 px-6 lg:px-4 flex gap-16`}>
            <div className="flex flex-col gap-4">
              <header className="mt-2">
                <h1 className="text-5xl font-normal flex flex-col gap-2">
                  Patterns{" "}
                  <span className="text-xl text-slate-400">
                    Async + Event-Driven
                  </span>
                </h1>
              </header>
              <p className="my-4 text-slate-600">
                Building with events sometimes requires a different way to look
                at the problem & solution. These common patterns walk through
                what the solutions look like with or without using events.
              </p>
            </div>
            <div style={{ maxWidth: "30%" }} className="hidden sm:block">
              <img
                src="/assets/patterns/patterns-hero.png"
                className="max-w-full rounded-lg"
              />
            </div>
          </div>
        </div>

        {/* Background styles */}
        <section>
          {/* Content layout */}
          <div className={`${sectionClasses} my-14 px-6 lg:px-4`}>
            {SECTIONS.map((s, idx) => (
              <div key={s.title} className="my-8">
                <div className="flex">
                  <div className="w-11 text-2xl font-bold text-slate-400">
                    {zeroPad(idx + 1)}
                  </div>
                  <h2 className="text-2xl">{s.title}</h2>
                </div>
                <div className="ml-11 my-6 flex flex-col gap-6">
                  {s.articles.map(({ title, subtitle, tags, slug }) => (
                    <Link key={slug} href={`/patterns/${slug}`} passHref>
                      <a className="flex flex-col text-almost-black">
                        <h2 className="text-lg text-color-dark-purple">
                          {title}
                        </h2>
                        <p className="text-sm mt-1 mb-3">{subtitle}</p>
                        <div className="flex gap-2">
                          {tags.map((t) => (
                            <span
                              key={t}
                              className="py-1 px-2 rounded-full bg-slate-100 text-slate-600"
                              style={{ fontSize: "0.6rem" }}
                            >
                              {t}
                            </span>
                          ))}
                        </div>
                      </a>
                    </Link>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>
      </Content>
      <Footer />
    </Page>
  );
}

export const Page = styled.div`
  background: radial-gradient(
    circle at 34% 290px,
    rgb(223 217 229 / 70%) 0,
    transparent 12%,
    transparent 100%
  );
  background-repeat: no-repeat;
`;

export const Content = styled.div`
  --font: "Inter", Helvetica, "SF Pro Display", -apple-system,
    BlinkMacSystemFont, Roboto, Open Sans, sans-serif;
  --font-heading: var(--font);
  font-family: var(--font);
`;
