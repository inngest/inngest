import styled from "@emotion/styled";
import Head from "next/head";
import Image from "next/image";
import rehypeSlug from "rehype-slug";
import rehypeRaw from "rehype-raw";
import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import Footer from "../../shared/Footer";
import { rehypePrependCode, rehypeShiki } from "../../utils/code";
import { rehypeParseCodeBlocks } from "../../mdx/rehype.mjs";
import Tags from "../../shared/Blog/Tags";

// MDX Components
import DiscordCTA from "../../shared/Blog/DiscordCTA";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Button from "src/shared/Button";
import IconCalendar from "src/shared/Icons/Calendar";

import * as mdxComponents from "../../shared/NewDocs/mdx";

const components = {
  DiscordCTA,
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
};

export default function BlogLayout(props) {
  const scope = JSON.parse(props.docs.scope.json);

  const structuredData = {
    "@context": "https://schema.org",
    "@type": "BlogPosting",
    headline: scope.heading,
    description: scope.subtitle,
    image: [`${process.env.NEXT_PUBLIC_HOST}${scope.image}`],
    datePublished: scope.date,
    dateModified: scope.date,
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

      <div className="bg-slate-1000 font-sans">
        <div
          style={{
            background: "radial-gradient(circle at center, #13123B, #08090d)",
          }}
          className="absolute w-[200vw] -translate-x-1/2 -translate-y-1/2 h-[200vw] rounded-full blur-lg opacity-90"
        ></div>
        <Header />
        <Container>
          <article>
            <main className="m-auto max-w-3xl pt-16">
              <header className="pt-12 lg:pt-24 max-w-[65ch] m-auto">
                <h1 className="text-white font-medium text-2xl md:text-4xl xl:text-5xl mb-2 md:mb-4 tracking-tighter lg:leading-loose">
                  {scope.heading}
                </h1>
              </header>
              <div className="m-auto mb-20 prose prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert">
                <MDXRemote
                  compiledSource={props.docs.compiledSource}
                  scope={scope}
                  components={{ mdxComponents }}
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
  const paths = fs.readdirSync("./pages/new-docs/_docs/").map((fname) => {
    return `/new-docs/${fname.replace(/.mdx?/, "")}`;
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

  let filePath = `./pages/new-docs/_docs/${params.slug}.md`;
  if (!fs.existsSync(filePath) && fs.existsSync(filePath + "x")) {
    filePath += "x";
  }

  const source = fs.readFileSync(filePath);
  const { content, data } = matter(source);

  data.path = `/new-docs/${params.slug}`;
  // data.reading = readingTime(content);
  // // Format the reading date.
  // data.humanDate = data.date.toLocaleDateString();

  // data.tags =
  //   data.tags && typeof data.tags === "string"
  //     ? data.tags.split(",").map((tag) => tag.trim())
  //     : data.tags;

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
  const docs = await serialize(content, {
    scope: { json: JSON.stringify(data) },
    mdxOptions: {
      rehypePlugins: [
        rehypeParseCodeBlocks,
        rehypePrependCode,
        rehypeShiki,
        [rehypeRaw, { passThrough: nodeTypes }],
        rehypeSlug,
      ],
    },
  });
  return {
    props: {
      docs,
      meta: {
        disabled: true,
      },
      designVersion: "2",
    },
  };
}
