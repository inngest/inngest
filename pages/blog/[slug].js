import styled from "@emotion/styled";
import Head from "next/head";
import rehypeSlug from "rehype-slug";
import { serialize } from "next-mdx-remote/serialize";
import { MDXRemote } from "next-mdx-remote";
import Footer from "../../shared/footer";
import Nav from "../../shared/nav";
import Callout from "../../shared/Callout";
import { Wrapper } from "../../shared/blog";
import { highlight } from "../../utils/code";

export default function BlogLayout(props) {
  const scope = JSON.parse(props.post.scope);
  return (
    <>
      <Head>
        <title>{scope.heading} → Inngest Blog</title>
      </Head>
      <Wrapper>
        <Nav sticky={true} />

        {scope.img && (
          <div className="image grid">
            <div className="header col-6-center sm-col-8-center">
              <Image>
                {scope.img && <img src={scope.img} alt="" className="hero" />}
              </Image>
            </div>
          </div>
        )}

        <Header className="grid">
          <header className="header col-6-center sm-col-8-center">
            <h1>{scope.heading}</h1>
            <p className="blog--date">
              {scope.humanDate} &middot; {scope.reading.text}
            </p>
          </header>
        </Header>
        <Main className="grid">
          <main className="col-6-center sm-col-8-center">
            <MDXRemote
              compiledSource={props.post.compiledSource}
              scope={scope}
            />
          </main>
        </Main>
        <Callout
          small="What is Inngest?"
          heading="The fastest way to build and ship event-driven functions"
          link="/?ref=blog-footer"
          cta="Learn more >"
        />
        <Footer />
      </Wrapper>
    </>
  );
}

const Header = styled.div`
  header {
    padding: var(--section-padding) 0;
  }
  .grid-line {
    padding: var(--section-padding) 0 0;
  }

  @media (max-width: 980px) {
    header {
      padding: 6vh 0;
    }
  }
`;

const Main = styled.div`
  > main {
    margin: var(--section-padding) 0 0;
  }

  video {
    margin: 4rem 0;
  }

  main {
    max-width: 980px;
    margin: 0 auto 4rem;
  }

  p {
    margin-bottom: 0.25rem;
  }
  ul {
    margin: 2rem 0 0;
  }

  h1 {
    margin: 0 0 2rem;
  }

  h2 {
    margin: 3.5rem 0 1rem;
    font-weight: 600;
    font-size: 1.65rem;
  }

  .blog--date {
    font-size: 0.85rem;
    opacity: 0.6;
    margin: -3.5rem 0 5rem;
    padding: 0;
    border-left: 2px solid var(--light-grey);
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

  img {
    max-width: 100%;
    max-height: 300px;
    margin: 3rem auto 3rem;
    pointer-events: none;
  }

  img.hero {
    padding: 0 0 50px;
  }

  code {
    background: rgb(46, 52, 64);
    padding: 0.1em 0.3em 0.15em;
    border-radius: 3px;
    color: rgb(216, 222, 233);
  }

  pre {
    margin: 3rem 0;
    padding: .25rem 1em;
    border-radius: 3px;
  }

  pre.shiki {
    overflow-x: auto;
  }
  pre.shiki div.dim {
    opacity: 0.8;
    transition: all 0.3s;
    &:hover {
      opacity: 1;
    }
  }
  pre.shiki div.dim,
  pre.shiki div.highlight {
    margin: 0;
    padding: 0;
  }
  pre.shiki div.highlight {
    opacity: 1;
    background-color: rgba(255, 255, 255, 0.05);
  }
  pre.shiki div.line {
    min-height: 1rem;
  }

  /** Don't show the language identifiers */
  pre.shiki .language-id {
    display: none;
  }

  /* Visually differentiates twoslash code samples  */
  pre.twoslash {
    border-color: #719af4;
  }

  /** When you mouse over the pre, show the underlines */
  pre.twoslash:hover data-lsp {
    border-color: #747474;
  }

  /** The tooltip-like which provides the LSP response */
  pre.twoslash data-lsp:hover::before {
    content: attr(lsp);
    position: absolute;
    transform: translate(0, 1rem);

    background-color: #3f3f3f;
    color: #fff;
    text-align: left;
    padding: 5px 8px;
    border-radius: 2px;
    font-family: "JetBrains Mono", Menlo, Monaco, Consolas, Courier New,
      monospace;
    font-size: 14px;
    white-space: pre-wrap;
    z-index: 100;
  }

  pre .code-container {
    overflow: auto;
  }
  /* The try button */
  pre .code-container > a {
    position: absolute;
    right: 8px;
    bottom: 8px;
    border-radius: 4px;
    border: 1px solid #719af4;
    padding: 0 8px;
    color: #719af4;
    text-decoration: none;
    opacity: 0;
    transition-timing-function: ease;
    transition: opacity 0.3s;
  }
  /* Respect no animations */
  @media (prefers-reduced-motion: reduce) {
    pre .code-container > a {
      transition: none;
    }
  }
  pre .code-container > a:hover {
    color: white;
    background-color: #719af4;
  }
  pre .code-container:hover a {
    opacity: 1;
  }

  pre code {
    font-size: 15px;
    font-family: "JetBrains Mono", Menlo, Monaco, Consolas, Courier New,
      monospace;
    white-space: pre;
    -webkit-overflow-scrolling: touch;
  }
  pre code a {
    text-decoration: none;
  }
  pre data-err {
    /* Extracted from VS Code */
    background: url("data:image/svg+xml,%3Csvg%20xmlns%3D'http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg'%20viewBox%3D'0%200%206%203'%20enable-background%3D'new%200%200%206%203'%20height%3D'3'%20width%3D'6'%3E%3Cg%20fill%3D'%23c94824'%3E%3Cpolygon%20points%3D'5.5%2C0%202.5%2C3%201.1%2C3%204.1%2C0'%2F%3E%3Cpolygon%20points%3D'4%2C0%206%2C2%206%2C0.6%205.4%2C0'%2F%3E%3Cpolygon%20points%3D'0%2C2%201%2C3%202.4%2C3%200%2C0.6'%2F%3E%3C%2Fg%3E%3C%2Fsvg%3E")
      repeat-x bottom left;
    padding-bottom: 3px;
  }
  pre .query {
    margin-bottom: 10px;
    color: #137998;
    display: inline-block;
  }

  /* In order to have the 'popped out' style design and to not break the layout
  /* we need to place a fake and un-selectable copy of the error which _isn't_ broken out
  /* behind the actual error message.

  /* This sections keeps both of those two in in sync  */

  pre .error,
  pre .error-behind {
    margin-left: -14px;
    margin-top: 8px;
    margin-bottom: 4px;
    padding: 6px;
    padding-left: 14px;
    width: calc(100% - 20px);
    white-space: pre-wrap;
    display: block;
  }
  pre .error {
    position: absolute;
    background-color: #fee;
    border-left: 2px solid #bf1818;
    /* Give the space to the error code */
    display: flex;
    align-items: center;
    color: black;
  }
  pre .error .code {
    display: none;
  }
  pre .error-behind {
    user-select: none;
    visibility: transparent;
    color: #fee;
  }
  /* Queries */
  pre .arrow {
    /* Transparent background */
    background-color: #eee;
    position: relative;
    top: -7px;
    margin-left: 0.1rem;
    /* Edges */
    border-left: 1px solid #eee;
    border-top: 1px solid #eee;
    transform: translateY(25%) rotate(45deg);
    /* Size */
    height: 8px;
    width: 8px;
  }
  pre .popover {
    margin-bottom: 10px;
    background-color: #eee;
    display: inline-block;
    padding: 0 0.5rem 0.3rem;
    margin-top: 10px;
    border-radius: 3px;
  }
  /* Completion */
  pre .inline-completions ul.dropdown {
    display: inline-block;
    position: absolute;
    width: 240px;
    background-color: gainsboro;
    color: grey;
    padding-top: 4px;
    font-family: var(--code-font);
    font-size: 0.8rem;
    margin: 0;
    padding: 0;
    border-left: 4px solid #4b9edd;
  }
  pre .inline-completions ul.dropdown::before {
    background-color: #4b9edd;
    width: 2px;
    position: absolute;
    top: -1.2rem;
    left: -3px;
    content: " ";
  }
  pre .inline-completions ul.dropdown li {
    overflow-x: hidden;
    padding-left: 4px;
    margin-bottom: 4px;
  }
  pre .inline-completions ul.dropdown li.deprecated {
    text-decoration: line-through;
  }
  pre .inline-completions ul.dropdown li span.result-found {
    color: #4b9edd;
  }
  pre .inline-completions ul.dropdown li span.result {
    width: 100px;
    color: black;
    display: inline-block;
  }
  .dark-theme .markdown pre {
    background-color: #d8d8d8;
    border-color: #ddd;
    filter: invert(98%) hue-rotate(180deg);
  }
  data-lsp {
    /* Ensures there's no 1px jump when the hover happens */
    border-bottom: 1px dotted transparent;
    /* Fades in unobtrusively */
    transition-timing-function: ease;
    transition: border-color 0.3s;
  }
  /* Respect people's wishes to not have animations */
  @media (prefers-reduced-motion: reduce) {
    data-lsp {
      transition: none;
    }
  }

  /** Annotations support, providing a tool for meta commentary */
  .tag-container {
    position: relative;
  }
  .tag-container .twoslash-annotation {
    position: absolute;
    font-family: "JetBrains Mono", Menlo, Monaco, Consolas, Courier New,
      monospace;
    right: -10px;
    /** Default annotation text to 200px */
    width: 200px;
    color: #187abf;
    background-color: #fcf3d9 bb;
  }
  .tag-container .twoslash-annotation p {
    text-align: left;
    font-size: 0.8rem;
    line-height: 0.9rem;
  }
  .tag-container .twoslash-annotation svg {
    float: left;
    margin-left: -44px;
  }
  .tag-container .twoslash-annotation.left {
    right: auto;
    left: -200px;
  }
  .tag-container .twoslash-annotation.left svg {
    float: right;
    margin-right: -5px;
  }

  /** Support for showing console log/warn/errors inline */
  pre .logger {
    display: grid;
    grid-template-columns: 15px 1fr;
    grid-gap: 10px;

    align-items: center;
    color: black;
    padding: 4px 6px;
    padding-left: 8px;
    width: calc(100% - 19px);
    white-space: pre-wrap;
    font-size: 12px;
    margin: 5px 0 10px;
    opacity: 0.3;
    cursor: default;
    transition: all 0.3s;
    &:hover {
      opacity: 0.8;
    }
  }
  pre .logger.error-log {
    background-color: #fee;
    border-left: 2px solid #bf1818;
  }
  pre .logger.warn-log {
    background-color: #ffe;
    border-left: 2px solid #eae662;
  }
  pre .logger.log-log {
    background: #090a12;
    border-left: 2px solid #ababab;
    color: #fff;
  }
  pre .logger.log-log svg {
    margin-left: 6px;
    margin-right: 9px;
    height: 10px;
  }

  @media (max-width: 980px) {
    .blog--callout {
      padding: 1.5rem;
      width: calc(100% - 1rem);
      margin-left: -1rem;
    }
  }
`;

const Image = styled.div`
  text-align: center;
  padding: 4em 0 0;

  display: flex;
  align-items: center;
  justify-content: center;

  img {
    max-height: 25vh;
  }
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

  const source = fs.readFileSync("./pages/blog/_posts/" + params.slug + ".md");
  const { content, data } = matter(source);

  data.reading = readingTime(content);
  // Format the reading date.
  data.humanDate = data.date.toLocaleDateString();

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
