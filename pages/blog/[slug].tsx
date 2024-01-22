import styled from "@emotion/styled";
import Head from "next/head";
import Image from "next/image";
import rehypeSlug from "rehype-slug";
import rehypeRaw from "rehype-raw";
import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import Footer from "../../shared/Footer";
import { rehypeRemoveTwoSlashMarkup, rehypeShiki } from "../../utils/code";
import { rehypeParseCodeBlocks } from "../../mdx/rehype.mjs";
import Tags from "../../shared/Blog/Tags";

// MDX Components
import DiscordCTA from "../../shared/Blog/DiscordCTA";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import { Button } from "src/shared/Button";
import IconCalendar from "src/shared/Icons/Calendar";
import CTACallout from "src/shared/CTACallout";
import Blockquote from "src/shared/Blog/Blockquote";
import rehypeCodeTitles from "rehype-code-titles";
import YouTube from "react-youtube-embed";
import remarkGfm from "remark-gfm";
import { LaunchWeekBanner } from "../index";

const components = {
  DiscordCTA,
  Button,
  CTACallout,
  Blockquote,
  YouTube,
};

type Props = {
  post: {
    compiledSource: string;
    scope: {
      json: string;
    };
  };
  meta: {
    disabled: true;
  };
};

const authorURLs = {
  "Dan Farrelly": "https://twitter.com/djfarrelly",
  "Tony Holdstock-Brown": "https://twitter.com/itstonyhb",
  "Jack Williams": "https://twitter.com/atticjack",
  "Igor Gassmann": "https://twitter.com/i_gassmann",
  "Darwin Wu": "https://twitter.com/67darwin",
  "Joel Hooks": "https://twitter.com/jhooks",
};

export default function BlogLayout(props) {
  const scope = JSON.parse(props.post.scope.json);

  const structuredData = {
    "@context": "https://schema.org",
    "@type": "BlogPosting",
    headline: scope.heading,
    description: scope.subtitle,
    image: [`${process.env.NEXT_PUBLIC_HOST}${scope.image}`],
    datePublished: scope.date,
    dateModified: scope.date,
    introCallout: scope.introCallout,
    author: [
      {
        "@type": scope.author ? "Person" : "Organization",
        name: scope.author || "Inngest",
        url:
          scope.author && authorURLs.hasOwnProperty(scope.author)
            ? authorURLs[scope.author]
            : process.env.NEXT_PUBLIC_HOST,
      },
    ],
  };
  const title = `${scope.heading} - Inngest Blog`;

  return (
    <>
      <Head>
        <title>{title}</title>
        <meta name="description" content={scope.subtitle}></meta>
        <meta name="title" content={scope.heading}></meta>
        <meta property="og:title" content={`${scope.heading} - Inngest Blog`} />
        <meta property="og:description" content={scope.subtitle} />
        <meta property="og:type" content="article" />
        <meta
          property="og:url"
          content={`${process.env.NEXT_PUBLIC_HOST}${scope.path}`}
        />
        {!!scope.image && (
          <meta
            property="og:image"
            content={`${process.env.NEXT_PUBLIC_HOST}${scope.image}`}
          />
        )}
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:site" content="@inngest" />
        <meta
          name="twitter:title"
          content={`${scope.heading} - Inngest Blog`}
        />
        <meta name="twitter:description" content={scope.subtitle} />
        {!!scope.image && (
          <meta
            name="twitter:image"
            content={`${process.env.NEXT_PUBLIC_HOST}${scope.image}`}
          />
        )}
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify(structuredData),
          }}
        ></script>
      </Head>

      {/* <ThemeToggleButton isFloating={true} /> */}

      <div className="font-sans">
        <Header />
        <LaunchWeekBanner urlRef="blog-post-banner" />
        <Container>
          <article>
            <main className="m-auto max-w-3xl pt-16">
              {scope.image && (
                <figure className="mx-auto flex flex-col items-end max-w-[768px]">
                  <Image
                    className="rounded-lg shadow-lg"
                    src={scope.image}
                    alt={`Featured image for ${scope.heading} blog post`}
                    width={768}
                    height={768 / 2}
                    quality={95}
                  />
                  {scope.imageCredits && (
                    <figcaption
                      className="text-xs text-slate-400 mt-2"
                      dangerouslySetInnerHTML={{ __html: scope.imageCredits }}
                    ></figcaption>
                  )}
                </figure>
              )}
              <header className="pt-12 lg:pt-24 max-w-[70ch] m-auto">
                <h1 className="text-white font-medium text-2xl md:text-4xl xl:text-5xl mb-2 md:mb-4 tracking-tighter lg:leading-loose">
                  {scope.heading}
                </h1>
                {scope.showSubtitle && (
                  <p className="text-slate-200 text-lg font-bold mb-6 flex gap-1 items-center">
                    {scope.subtitle}
                  </p>
                )}
                <p className="text-slate-300 text-sm mt-2 flex items-center gap-2">
                  {!!scope.author ? <>{scope.author} &middot; </> : ""}
                  <span className="flex items-center gap-1">
                    <IconCalendar /> {scope.humanDate}
                  </span>{" "}
                  &middot; <span>{scope.reading.text}</span>
                  <Tags tags={scope.tags} />
                </p>
              </header>
              {scope.introCallout && (
                <CTACallout
                  text={scope.introCallout}
                  cta={{
                    href: "https://www.inngest.com?ref=blog-post",
                    text: "Give it a try",
                  }}
                />
              )}
              {!scope.disableCTA && !scope.introCallout && (
                <CTACallout
                  text={
                    <>
                      <a
                        className="text-indigo-400 font-medium hover:text-white transition-all no-underline hover:underline"
                        href="https://www.inngest.com?ref=blog-post"
                      >
                        Inngest
                      </a>{" "}
                      is the developer platform for easily building reliable
                      workflows and background jobs with zero infrastructure.
                    </>
                  }
                  cta={{
                    href: "https://www.inngest.com?ref=blog-post",
                    text: "Give it a try",
                  }}
                />
              )}
              <div className="max-w-[70ch] prose m-auto mt-12 mb-20 prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert blog-content">
                <MDXRemote
                  compiledSource={props.post.compiledSource}
                  scope={scope}
                  components={components}
                />
              </div>
              <DiscordCTA />
            </main>
          </article>
        </Container>
        <Footer />
      </div>
    </>
  );
}

// This function gets called at build time to figure out which URLs
// we need to statically compile.
//
// These URLs will be treated as individual pages. getStaticProps is
// called for each URL with the slug in params.
export async function getStaticPaths() {
  const fs = require("fs");
  const paths = fs.readdirSync("./pages/blog/_posts/").map((fname) => {
    return `/blog/${fname.replace(/.mdx?/, "")}`;
  });
  return { paths, fallback: false };
}

// This function also gets called at build time to generate specific content.
export async function getStaticProps({ params }) {
  // These are required here as this function is not included in frontend
  // browser builds.
  const fs = require("fs");
  const readingTime = require("reading-time");
  const matter = require("gray-matter");

  let filePath = `./pages/blog/_posts/${params.slug}.md`;
  if (!fs.existsSync(filePath) && fs.existsSync(filePath + "x")) {
    filePath += "x";
  }

  const source = fs.readFileSync(filePath);
  const { content, data } = matter(source);

  data.path = `/blog/${params.slug}`;
  data.reading = readingTime(content);
  // Format the reading date.
  data.humanDate = data.date.toLocaleDateString();

  data.tags =
    data.tags && typeof data.tags === "string"
      ? data.tags.split(",").map((tag) => tag.trim())
      : data.tags;

  // type Post = {
  //   compiledSource: string,
  //   scope: string,
  // }
  const nodeTypes = [
    "mdxFlowExpression",
    "mdxJsxFlowElement",
    "mdxJsxTextElement",
    "mdxTextExpression",
    "mdxjsEsm",
  ];
  const post = await serialize(content, {
    scope: { json: JSON.stringify(data) },
    mdxOptions: {
      rehypePlugins: [
        rehypeCodeTitles,
        rehypeParseCodeBlocks,
        rehypeRemoveTwoSlashMarkup,
        rehypeShiki,
        [rehypeRaw, { passThrough: nodeTypes }],
        rehypeSlug,
        remarkGfm,
      ],
    },
  });
  return {
    props: {
      post,
      meta: {
        disabled: true,
      },
      designVersion: "2",
    },
  };
}
