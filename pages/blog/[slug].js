import styled from "@emotion/styled";
import Head from "next/head";
import rehypeSlug from "rehype-slug";
import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import Footer from "../../shared/footer";
import Nav from "../../shared/nav";
import Callout from "../../shared/Callout";
import syntaxHighlightingCSS from "../../shared/syntaxHighlightingCSS";
import { Wrapper } from "../../shared/blog";
import { highlight } from "../../utils/code";
import ThemeToggleButton from "../../shared/ThemeToggleButton";
import Tags from "../../shared/Blog/Tags";

// MDX Components
import DiscordCTA from "../../shared/Blog/DiscordCTA";
const components = {
  DiscordCTA,
};

export default function BlogLayout(props) {
  const scope = JSON.parse(props.post.scope);
  return (
    <>
      <Head>
        <title>{scope.heading} → Inngest Blog</title>
        <meta name="description" content={scope.subtitle}></meta>
        <meta property="og:title" content={`${scope.heading} → Inngest Blog`} />
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
      </Head>

      <ThemeToggleButton isFloating={true} />

      <Wrapper>
        <Nav sticky={true} />

        <Article>
          {scope.image && <Image src={scope.image} />}

          <Header>
            <h1>{scope.heading}</h1>
            <p className="blog-byline">
              {!!scope.author ? <>{scope.author} &middot; </> : ""}
              {scope.humanDate} &middot; {scope.reading.text}
              <Tags tags={scope.tags} />
            </p>
          </Header>

          <Body>
            <MDXRemote
              compiledSource={props.post.compiledSource}
              scope={scope}
              components={components}
            />
          </Body>
          <DiscordCTA />
        </Article>
        <Footer />
      </Wrapper>
    </>
  );
}

const Article = styled.article`
  margin: 1rem auto 4rem;
  max-width: 800px;

  @media (max-width: 800px) {
    margin-left: 1.5rem;
    margin-right: 1.6rem;
  }
`;

const Image = styled.img`
  margin: 1rem auto;
  max-width: 100%;
  border-radius: var(--border-radius);
`;

const Header = styled.header`
  margin: 3rem 0;
  h1 {
    font-size: 2rem;
    line-height: 1.5em;
  }
  .blog-byline {
    font-size: 0.8rem;
  }
  @media (max-width: 800px) {
    margin: 2rem 0;
    h1 {
      font-size: 1.8rem;
      line-height: 1.3em;
    }
  }
`;

const Body = styled.main`
  margin: 2rem 0;

  h2,
  h3,
  h4,
  h5 {
    line-height: 1.5em;
  }
  h2 {
    font-size: 1.5em;
    margin-top: 2rem;
  }
  h3 {
    font-size: 1.3em;
    margin-top: 2rem;
  }
  h3 {
    font-size: 1.3em;
    margin-top: 2rem;
  }

  p,
  blockquote {
    margin: 1.5rem 0;
    line-height: 1.6em;
  }

  ol,
  ul {
    margin: 1.5rem 0;
  }
  li {
  }

  video {
    margin: 2rem 0;
  }

  img {
    max-width: 100%;
    /* max-height: 300px; */
    margin: 2rem auto 2rem;
    pointer-events: none;
  }

  blockquote {
    padding: 0 1.5rem;
    border-left: 4px solid var(--primary-color);
    font-style: italic;
  }

  p code,
  li code {
    background: rgb(46, 52, 64);
    padding: 0.1em 0.3em 0.15em;
    border-radius: 3px;
    color: rgb(216, 222, 233);
  }

  a:not(.button) {
    color: var(--color-iris-60);
  }

  pre {
    margin: 1rem 0;
    padding: 1.1rem 1.4rem;
    border-radius: var(--border-radius);
  }

  .blog--callout {
    font-weight: 500;

    box-sizing: content-box;
    padding: 2rem;
    margin: 0 0 8vh;
    border-radius: 10px;

    background-image: linear-gradient(
      -45deg,
      rgba(255, 255, 255, 0.08) 25%,
      transparent 25%,
      transparent 50%,
      rgba(255, 255, 255, 0.08) 50%,
      rgba(255, 255, 255, 0.08) 75%,
      transparent 75%,
      transparent
    );
    background-size: 5px 5px;
  }

  @media (max-width: 800px) {
    p,
    ol,
    ul,
    li {
      font-size: 0.8rem;
    }
    h1 {
      font-size: 1.8rem;
    }
  }

  @media (max-width: 980px) {
    .blog--callout {
      padding: 1.5rem;
      width: calc(100% - 1rem);
      margin-left: -1rem;
    }
  }

  ${syntaxHighlightingCSS}
`;

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
  const post = await serialize(content, {
    scope: JSON.stringify(data),
    mdxOptions: {
      remarkPlugins: [highlight],
      rehypePlugins: [rehypeSlug],
    },
  });
  return { props: { post } };
}
