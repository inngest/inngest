import Head from 'next/head';
import Image from 'next/image';
import { useRouter } from 'next/router';
import styled from '@emotion/styled';
import { Rss } from 'react-feather';
import ArrowRight from 'src/shared/Icons/ArrowRight';
import IconCalendar from 'src/shared/Icons/Calendar';
import SectionHeader from 'src/shared/SectionHeader';

import Tags from '../shared/Blog/Tags';
import Footer from '../shared/Footer';
import Header from '../shared/Header';
import Container from '../shared/layout/Container';
import ThemeToggleButton from '../shared/legacy/ThemeToggleButton';
import Nav from '../shared/legacy/nav';
import { loadMarkdownFilesMetadata, type MDXFileMetadata } from '../utils/markdown';
import { LaunchWeekBanner } from './index';

export default function BlogLayout(props) {
  const router = useRouter();
  const { showHidden } = router.query;

  const content: BlogPost[] = props.content.map(JSON.parse);
  const visiblePosts = showHidden
    ? content
    : content.filter((post) => !post.hide).sort((a, z) => z.date.localeCompare(a.date));

  const focus = visiblePosts.find((c) => c.focus) ?? visiblePosts[0];
  const rest = visiblePosts
    .filter((c) => !focus || c.slug !== focus.slug)
    .sort((a, z) => z.date.localeCompare(a.date));

  const description = `Updates from the Inngest team about our product, engineering, and community.`;

  return (
    <>
      <Head>
        <title>Inngest → Product & Engineering blog</title>
        <meta name="description" content={description}></meta>
        <meta property="og:title" content="Inngest → Product & Engineering blog" />
        <meta property="og:description" content={description} />
      </Head>

      <div className="font-sans">
        <Header />

        <LaunchWeekBanner urlRef="blog-feed-banner" />

        <Container className="pt-8">
          <div className="flex flex-col items-start gap-2 lg:flex-row lg:items-center lg:gap-4">
            <h2 className="border-slate-600/50 pr-4 text-base font-bold text-white lg:border-r">
              Blog
            </h2>
            <p className="text-sm text-slate-200">{description}</p>
            <a
              href="/api/rss.xml"
              className="rounded-md border border-transparent py-1 text-slate-300 transition-all hover:border-slate-200/30 hover:text-white"
            >
              <Rss className="h-4" />
            </a>
          </div>
          <div className="pt-16">
            {focus && (
              <a
                className="group relative mb-32 flex flex-col-reverse rounded-lg bg-indigo-600 shadow-lg lg:flex-row xl:max-w-[1160px]"
                href={focus.redirect ?? `/blog/${focus.slug}`}
              >
                <div className="absolute -left-[40px] -right-[40px] bottom-0 top-0 -z-0 mx-5 rotate-1 rounded-lg bg-indigo-500 opacity-20"></div>
                <div className="relative z-10 flex flex-col items-start justify-between p-8 lg:w-2/5">
                  <div>
                    <span className="mb-3 inline-flex rounded bg-indigo-700/50 px-3 py-1.5 text-xs font-semibold text-indigo-50">
                      Latest Post
                    </span>
                    <h2 className="mb-1 text-xl font-medium text-white md:text-2xl lg:text-xl xl:text-2xl">
                      {focus.heading}
                    </h2>
                    <p className="mb-4 flex items-center gap-1 text-sm font-medium text-slate-200">
                      <IconCalendar />
                      {focus.humanDate} <Tags tags={focus.tags} />
                    </p>
                    <p className="text-slate-100">{focus.subtitle}</p>
                  </div>
                  <span className="mt-4 inline-flex rounded-full bg-slate-800 px-4 py-1.5 text-sm font-medium text-slate-50 group-hover:bg-slate-700">
                    Read article
                    <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
                  </span>
                </div>
                {focus.image && (
                  <div className="relative flex overflow-hidden rounded-t-lg transition-all group-hover:scale-105 group-hover:rounded-lg lg:w-3/5 lg:rounded-r-lg lg:rounded-t-none">
                    <Image
                      className="z-10 m-auto w-full rounded-t-lg group-hover:rounded-lg lg:rounded-r-lg lg:rounded-t-none"
                      src={focus.image}
                      alt={`Featured image for ${focus.heading} blog post`}
                      width={900}
                      height={900 / 2}
                      quality={95}
                    />
                    <Image
                      className="w-[calc(100% + theme('spacing.4'))] absolute -bottom-2 -left-2 -right-2 -top-2 z-0 m-auto h-[110%] w-full max-w-none rounded-t-lg opacity-90 blur-sm lg:rounded-r-lg lg:rounded-t-none"
                      src={focus.image}
                      alt={`Featured image for ${focus.heading} blog post`}
                      width={900}
                      height={900 / 2}
                      quality={95}
                    />
                  </div>
                )}
              </a>
            )}

            <ul className="grid grid-cols-1 gap-x-8 gap-y-20 md:grid-cols-2 lg:grid-cols-3 lg:gap-x-4  xl:gap-x-8">
              {rest.map((item) => (
                <li key={item.slug}>
                  <a
                    href={item.redirect ?? `/blog/${item.slug}`}
                    className="group flex flex-col rounded-lg transition-all ease-out "
                  >
                    {item.image && (
                      <div className="flex rounded-lg shadow transition-all group-hover:scale-105">
                        {/* We use 720 as the responsive view goes full width at 720px viewport width */}
                        <Image
                          className="rounded-lg"
                          src={item.image}
                          alt={`Featured image for ${item.heading} blog post`}
                          width={720}
                          height={720 / 2}
                        />
                      </div>
                    )}
                    <div className="pt-4 xl:py-4 xl:pt-6">
                      <h2 className="mb-1 text-base text-white transition-all group-hover:text-indigo-400 xl:text-lg">
                        {item.heading}
                      </h2>
                      <p className="mb-4 mt-2 flex items-center gap-1 text-sm font-medium text-slate-400">
                        <IconCalendar />
                        {item.humanDate} <Tags tags={item.tags} />
                      </p>
                      <p className="text-sm text-slate-300">{item.subtitle}</p>
                    </div>
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </Container>
        <Footer />
      </div>
    </>
  );
}

export type BlogPost = {
  heading: string;
  subtitle: string;
  author?: string;
  image: string;
  date: string;
  humanDate: string;
  tags?: string[];
  hide?: boolean;
} & MDXFileMetadata;

// This function also gets called at build time to generate specific content.
export async function getStaticProps() {
  const posts = await loadMarkdownFilesMetadata<BlogPost>('blog/_posts');
  const content = posts.map((p) => JSON.stringify(p));

  return {
    props: {
      content,
      designVersion: '2',
      meta: {
        // TODO
        title: 'Product & Engineering Blog',
        description: `Updates from the Inngest team about our product, engineering, and community.`,
      },
    },
  };
}
