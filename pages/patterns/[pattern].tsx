import React from "react";
import styled from "@emotion/styled";
import { GetStaticProps, GetStaticPaths } from "next";
import Link from "next/link";
import { MDXRemote } from "next-mdx-remote";

import Nav from "src/shared/legacy/nav";
import Footer from "src/shared/legacy/Footer";
import { Heading } from "src/utils/docs";

import { loadMarkdownFile } from "utils/markdown";

import { SECTIONS, Page, Content } from "./index";

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
  const sectionClasses = `max-w-2xl mx-auto px-6 lg:px-4 text-left`;
  return (
    <Page>
      <Nav sticky={true} />

      <Content>
        {/* Background styles */}
        <div>
          {/* Content layout */}
          <div className={`${sectionClasses} mt-28 mb-14`}>
            <div className="flex flex-col gap-4">
              <header className="mt-2">
                <p className="text-xs font-normal flex gap-1">
                  <Link
                    href="/patterns"
                    className="text-almost-black transition-all duration-300 hover:-translate-x-0.5"
                  >
                    <span className="text-slate-400">←</span> Patterns{" "}
                    <span className="text-slate-400">Async + Event-Driven</span>
                  </Link>
                </p>

                <h1 className="mt-2 tracking-tight">{title}</h1>
              </header>
              <p className="my-2 text-slate-600">{subtitle}</p>
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
            </div>
          </div>
        </div>

        {/* Background styles */}
        <JumpLinks className={`${sectionClasses}`}>
          <h3 className="text-sm">Jump to</h3>
          <ol className="mt-2">
            {headings.map((h) => (
              <li key={h.slug} className="text-xs">
                <a href={`#${h.slug}`}>→ {h.title}</a>
              </li>
            ))}
          </ol>
        </JumpLinks>

        <Article>
          {/* Content layout */}
          <div className={`${sectionClasses} my-14`}>
            <MDXRemote compiledSource={compiledSource} components={{}} />

            <p className="mt-24 text-2xl font-normal flex gap-1">
              <Link
                href="/patterns"
                className="text-almost-black transition-all	duration-300 hover:-translate-x-1"
              >
                <span className="text-slate-400">←</span> View All Patterns
              </Link>
            </p>
          </div>
        </Article>
      </Content>
      <Footer />
    </Page>
  );
}

const JumpLinks = styled.aside`
  @media (min-width: calc(840px + 2 * (300px + 1rem))) {
    position: fixed;
    top: calc(
      32px + 87px + 7rem + 2rem
    ); // banner + nav + header margin(s) + backlink
    right: 1rem;
  }
`;

const Article = styled.article`
  // For vertical margin for scroll anchors
  h2,
  h3 {
    scroll-margin-top: 120px; /* Height of nav + margin */
  }

  h2 {
    margin: 1.5rem 0 1rem;
    font-size: 1.25rem;
    line-height: 1.75rem;
    letter-spacing: -0.025em;
  }
  ol,
  ul {
    margin: 1.5rem 0 1.5rem 1.5rem;
  }
  ol {
    list-style: decimal;
  }
  ul {
    list-style: circle;
  }
`;
