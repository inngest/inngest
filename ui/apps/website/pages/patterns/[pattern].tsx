import React from 'react';
import { GetStaticPaths, GetStaticProps } from 'next';
import Link from 'next/link';
import { MDXRemote } from 'next-mdx-remote';
import { Button } from 'src/shared/Button';
import DebugBreakpoints from 'src/shared/DebugBreakpoints';
import Container from 'src/shared/layout/Container';
import { Heading, loadMarkdownFile } from 'utils/markdown';

import Footer from '../../shared/Footer';
import Header from '../../shared/Header';
import * as MDXComponents from '../../shared/Patterns/mdx';
import { SECTIONS } from './index';

const getPatternProps = (slug: string) => {
  return SECTIONS.map((s) => s.articles)
    .flat()
    .find((a) => a.slug === slug);
};

export const getStaticProps: GetStaticProps = async (ctx) => {
  const slug = Array.isArray(ctx?.params?.pattern) ? ctx?.params?.pattern[0] : ctx?.params?.pattern;
  const pageInfo = getPatternProps(slug || '');
  const pageData = await loadMarkdownFile('patterns/_patterns', slug);
  return {
    props: {
      ...pageInfo,
      ...pageData,
      designVersion: '2',
      meta: {
        title: 'Patterns: Async + Event-Driven',
        description: 'A collection of software architecture patterns for asynchronous flows',
        image: '/assets/patterns/og-image-patterns.jpg',
      },
    },
  };
};

export const getStaticPaths: GetStaticPaths = async () => {
  const slugs = SECTIONS.map((s) => s.articles.map((a) => a.slug)).flat();
  // TEMP - filter only paths that have valid slugs
  const paths = slugs.filter((s) => s !== '#TODO').map((slug) => ({ params: { pattern: slug } }));
  return { paths, fallback: false };
};

type Props = {
  title: string;
  subtitle: string;
  tags: string[];
  headings: Heading[];
  compiledSource: string;
};

export default function Patterns({ title, subtitle, tags, headings, compiledSource }: Props) {
  return (
    <div className="relative">
      <Header />

      <div
        className="absolute bottom-0 left-0 right-0 top-0 z-0 m-auto opacity-20"
        style={{
          backgroundImage: 'url(/assets/patterns/hero-blur.png)',
          backgroundPosition: 'center 140px',
          backgroundRepeat: 'no-repeat',
          backgroundSize: '1800px 1200px',
        }}
      ></div>

      <Container className="pb-20 pt-12">
        <div className="m-auto max-w-[65ch] text-left lg:max-w-none">
          <header>
            <Button href="/patterns" variant="secondary" size="sm" arrow="left">
              Back to Patterns
            </Button>

            <h1 className="mt-8 text-3xl font-semibold tracking-tighter text-white sm:text-5xl">
              {title}
            </h1>
          </header>
          <p className=" mb-6 mt-2 max-w-[640px] text-base text-indigo-200 md:text-lg">
            {subtitle}
          </p>
          <div className="flex gap-2">
            {tags.map((t) => (
              <span
                key={t}
                className="rounded bg-slate-800 px-2 py-1 text-xs font-medium text-slate-300 transition-all group-hover/card:bg-slate-200 group-hover/card:text-slate-500"
              >
                {t}
              </span>
            ))}
          </div>
        </div>
      </Container>
      <div className="bg-slate-1000/80">
        <Container className="sm:pt-8 md:pt-12 lg:grid lg:grid-cols-3 lg:pt-20">
          <aside className="top-32 -mx-6 mb-12 max-w-[65ch] self-start bg-slate-500/20 p-6 pr-8 sm:mx-auto sm:rounded lg:sticky lg:col-start-4 lg:max-w-[320px] xl:max-w-[400px] xl:p-8 xl:pr-12 ">
            <h3 className="text-sm font-medium text-slate-400">Jump to</h3>
            <ol className="mt-2 flex flex-col gap-2">
              {headings.map((h) => (
                <li key={h.slug} className=" ">
                  <a
                    href={`#${h.slug}`}
                    className="text-sm font-medium leading-tight tracking-tight  text-white transition-all hover:underline "
                  >
                    {h.title}
                  </a>
                </li>
              ))}
            </ol>
          </aside>

          {/* <article className="col-span-3 row-start-1 col-start-1 xl:col-start-2 xl:col-span-3 max-w-[65ch] prose m-auto mb-20 prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert"> */}
          <article className="prose prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert m-auto mb-20 max-w-[65ch] text-slate-400 lg:col-span-3 lg:col-start-1 lg:row-start-1 lg:m-0 lg:max-w-none lg:pr-12 xl:pr-20">
            <MDXRemote
              compiledSource={compiledSource}
              components={MDXComponents}
              frontmatter={{}}
              scope={{}}
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
