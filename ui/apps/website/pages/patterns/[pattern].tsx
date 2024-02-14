import React from "react";
import { GetStaticProps, GetStaticPaths } from "next";
import Link from "next/link";
import { MDXRemote } from "next-mdx-remote";

import Container from "src/shared/layout/Container";
import Header from "../../shared/Header";
import Footer from "../../shared/Footer";
import { loadMarkdownFile, Heading } from "utils/markdown";
import * as MDXComponents from "../../shared/Patterns/mdx";
import { SECTIONS } from "./index";
import { Button } from "src/shared/Button";
import DebugBreakpoints from "src/shared/DebugBreakpoints";

const getPatternProps = (slug: string) => {
  return SECTIONS.map((s) => s.articles)
    .flat()
    .find((a) => a.slug === slug);
};

export const getStaticProps: GetStaticProps = async (ctx) => {
  const slug = Array.isArray(ctx?.params?.pattern)
    ? ctx?.params?.pattern[0]
    : ctx?.params?.pattern;
  const pageInfo = getPatternProps(slug || "");
  const pageData = await loadMarkdownFile("patterns/_patterns", slug);
  return {
    props: {
      ...pageInfo,
      ...pageData,
      designVersion: "2",
      meta: {
        title: "Patterns: Async + Event-Driven",
        description:
          "A collection of software architecture patterns for asynchronous flows",
        image: "/assets/patterns/og-image-patterns.jpg",
      },
    },
  };
};

export const getStaticPaths: GetStaticPaths = async () => {
  const slugs = SECTIONS.map((s) => s.articles.map((a) => a.slug)).flat();
  // TEMP - filter only paths that have valid slugs
  const paths = slugs
    .filter((s) => s !== "#TODO")
    .map((slug) => ({ params: { pattern: slug } }));
  return { paths, fallback: false };
};

type Props = {
  title: string;
  subtitle: string;
  tags: string[];
  headings: Heading[];
  compiledSource: string;
};

export default function Patterns({
  title,
  subtitle,
  tags,
  headings,
  compiledSource,
}: Props) {
  return (
    <div className="relative">
      <Header />

      <div
        className="top-0 left-0 right-0 m-auto bottom-0 absolute z-0 opacity-20"
        style={{
          backgroundImage: "url(/assets/patterns/hero-blur.png)",
          backgroundPosition: "center 140px",
          backgroundRepeat: "no-repeat",
          backgroundSize: "1800px 1200px",
        }}
      ></div>

      <Container className="pt-12 pb-20">
        <div className="text-left max-w-[65ch] m-auto lg:max-w-none">
          <header>
            <Button href="/patterns" variant="secondary" size="sm" arrow="left">
              Back to Patterns
            </Button>

            <h1 className="text-white font-semibold text-3xl mt-8 sm:text-5xl tracking-tighter">
              {title}
            </h1>
          </header>
          <p className=" text-indigo-200 text-base md:text-lg mt-2 mb-6 max-w-[640px]">
            {subtitle}
          </p>
          <div className="flex gap-2">
            {tags.map((t) => (
              <span
                key={t}
                className="py-1 px-2 rounded bg-slate-800 text-slate-300 group-hover/card:bg-slate-200 group-hover/card:text-slate-500 transition-all font-medium text-xs"
              >
                {t}
              </span>
            ))}
          </div>
        </div>
      </Container>
      <div className="bg-slate-1000/80">
        <Container className="lg:grid lg:grid-cols-3 sm:pt-8 md:pt-12 lg:pt-20">
          <aside className="max-w-[65ch] lg:max-w-[320px] xl:max-w-[400px] bg-slate-500/20 sm:rounded p-6 pr-8 xl:pr-12 xl:p-8 lg:sticky top-32 -mx-6 sm:mx-auto mb-12 lg:col-start-4 self-start ">
            <h3 className="text-sm text-slate-400 font-medium">Jump to</h3>
            <ol className="mt-2 flex flex-col gap-2">
              {headings.map((h) => (
                <li key={h.slug} className=" ">
                  <a
                    href={`#${h.slug}`}
                    className="text-white text-sm font-medium tracking-tight  hover:underline transition-all leading-tight "
                  >
                    {h.title}
                  </a>
                </li>
              ))}
            </ol>
          </aside>

          {/* <article className="col-span-3 row-start-1 col-start-1 xl:col-start-2 xl:col-span-3 max-w-[65ch] prose m-auto mb-20 prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert"> */}
          <article className="lg:col-span-3 lg:pr-12 xl:pr-20 lg:col-start-1 lg:row-start-1 max-w-[65ch] lg:max-w-none m-auto lg:m-0 prose mb-20 prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-400 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert">
            <MDXRemote
              compiledSource={compiledSource}
              components={MDXComponents}
            />
          </article>
          {/* <div className="col-start-2 col-span-3 max-w-[65ch]">
          <Button
            href="/patterns"
            variant="secondary"
            size="sm"
            arrow="left"
            className="col-start-2 place-self-start"
          >
            Back to Patterns
          </Button>
        </div> */}
        </Container>
      </div>

      <Footer />
    </div>
  );
}
