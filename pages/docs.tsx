import React from "react";
import Head from "next/head";
import { useRouter } from "next/router";
import styled from "@emotion/styled";

import Nav from "../shared/nav";
import Footer from "../shared/footer";
import { getAllDocs, Categories, Category, DocScope } from "../utils/docs";
import { categoryMeta } from "../utils/docsCategories";
import Home from "../shared/Icons/Home";

export default function DocsHome(props) {
  return (
    <DocsLayout categories={props.categories}>
      <Head>
        <title>Inngest → documentation & event-driven serverless guides</title>
      </Head>
      <Hero>
        <h1>Documentation</h1>
        {/* TODO: Quick start guide callouts, and graphic */}
      </Hero>

      <DocsContent>
        <div>
          <h2>What is Inngest</h2>
          <p>
            Inngest is a programmable event platform which allows you to
            aggregate every event in your business, and react to them by running
            code in real-time.
          </p>

          <p>
            We subscribe to every event in your stack, and allow you to run a
            DAG of serverless functions whenever specific events are received.
          </p>
          <p>
            Our platform allows you to build your product, ops, and internal
            flows behind a single abstraction: treating anything that happens
            across any service as a single event.
          </p>

          {/* Start building for free callout */}

          <h2>Discover Inngest</h2>

          <Discover>
            <div>
              <div>
                <h3>Getting Started</h3>
                <p>
                  A technical and non-technical introduction to the features of
                  Inngest, how it works, and step-by-step examples to get you
                  running in minutes.
                </p>
              </div>
              <ul>
                <li>
                  <a href="/docs/what-is-inngest">What is Inngest?</a>
                </li>
                <li>
                  <a href="/docs/how-inngest-works">How Inngest works</a>
                </li>
              </ul>
            </div>
          </Discover>
        </div>
      </DocsContent>
    </DocsLayout>
  );
}

export async function getStaticProps() {
  const categories = getAllDocs().categories;
  return { props: { categories } };
}

export const DocsLayout: React.FC<{ categories: Categories }> = ({
  children,
  categories,
}) => {
  const router = useRouter();
  return (
    <>
      <Nav />

      <div className="grid">
        <Menu className="col-3 sm-col-10">
          <h5>
            <span>Contents</span>
            <span
              className="toggle off"
              onClick={(e) => {
                const span = e.target as HTMLSpanElement;
                const on = !span.classList.contains("off");
                document.querySelectorAll(".category").forEach((a) => {
                  on
                    ? a.classList.remove("expanded")
                    : a.classList.add("expanded");
                });
                span.classList.toggle("off");
              }}
            >
              Toggle categories
            </span>
          </h5>

          <ul>
            <li>
              <a href="/docs" className="category" style={{ paddingBottom: 0 }}>
                <Home fill="#fff" size={20} /> Home
              </a>
            </li>
            {Object.values(categories).map((c) => {
              const meta = categoryMeta[c.title.toLowerCase()] || {};

              const isCurrent = !!c.pages.find(
                (p) => p.slug === router.asPath.replace("/docs/", "")
              );

              return (
                <li key={c.title}>
                  <span
                    className={["category", isCurrent && "expanded"]
                      .filter(Boolean)
                      .join(" ")}
                    onClick={(e) =>
                      (e.target as HTMLSpanElement).classList.toggle("expanded")
                    }
                  >
                    {meta.icon} {c.title}
                  </span>
                  <ul className="items">
                    {c.pages
                      .sort((a, b) => a.order - b.order)
                      .map((d) => renderDocLink(d, c, router.asPath))}
                  </ul>
                </li>
              );
            })}
          </ul>
        </Menu>
        <Content className="col-6 sm-col-10">{children}</Content>
      </div>
      <Footer />
    </>
  );
};

const renderDocLink = (s: DocScope, c: Category, currentRoute?: string) => {
  const currentSlug = currentRoute.replace(/^\/docs\//, "");

  // Find all pages with the given prefix.  These are the children.
  const children = c.pages.filter(
    (p) => p.slug.length > s.slug.length && p.slug.indexOf(s.slug) === 0
  );

  const id = s.title.toLowerCase().replace(/[^a-z]+/g, "-");

  // If we're on this page or any of the children under this page, we're "active"
  const isCurrent = !![s, ...children].find(
    (page) => page.slug.indexOf(currentSlug) === 0
  );

  if (s.slug.indexOf("/") > 0) {
    return null;
  }

  return (
    <li key={s.slug}>
      <a
        href={"/docs/" + s.slug}
        className={s.slug === currentSlug ? "active" : ""}
      >
        {s.title}
        {children.length > 0 && (
          <span
            className="toggle-subcategory"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              document.querySelector(`#${id}`).classList.toggle("expanded");
            }}
          >
            toggle
          </span>
        )}
      </a>
      {children.length > 0 && (
        <ul
          id={id}
          className={["subcategory", isCurrent && "expanded"]
            .filter(Boolean)
            .join(" ")}
        >
          {children.map((child) => {
            return (
              <li key={child.slug}>
                <a
                  href={"/docs/" + child.slug}
                  className={child.slug === currentSlug ? "active" : ""}
                >
                  {child.title}
                </a>
              </li>
            );
          })}
        </ul>
      )}
    </li>
  );
};

export const DocsContent = styled.div`
  display: grid;
  grid-template-columns: 3fr 1fr;

  h2 { margin-top: 4rem; }

  h3 {
    margin-top: 3rem;
  }

  /* "On this page" */
  h2 + h5 {
    margin-top: 3rem;
  }

  @media (max-width: 800px) {
    grid-template-columns: 1fr;

    h2 { margin-top: 2.5rem; }
  }
`;

export const InnerDocsContent = styled.div`
  padding-bottom: 10vh;

  h2 {
    padding-top: 2rem;
  }

  h4 {
    padding-top: 1.5rem;
  }

  .language-id {
    display: none;
  }

  ol,
  ul {
    margin: 1.4rem 0 1.5rem;
  }

  .tldr {
    border: 1px solid #ffffff33;
    border-radius: 3px;
    padding: 3rem 2rem 2rem;
    margin: 0 0 4rem;
    font-size: 0.9rem;
    position: relative;
    box-shadow: 0 5px 20px rgba(var(--black-rgb), 0.3);

    p,
    li {
      margin: 0;
    }
    p + p {
      margin: 1rem 0 0;
    }

    ol,
    ul {
      margin: 1rem 0;
    }

    &:before {
      content: "TL;DR";
      display: block;
      position: absolute;
      top: 1.3rem;
      left: 2rem;
      font-size: 0.9rem;
      opacity: 0.5;

      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 1px;
    }
  }

  img {
    max-width: 100%;
  }

  pre {
    margin: 3rem 0;
    padding: 1rem;
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
`;

const Menu = styled.div`
  border-right: 1px dashed var(--grid-line-color);
  border-bottom: 1px dashed var(--grid-line-color);

  padding: 3rem;
  background: rgba(0, 0, 0, 0.4);

  h5 {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .toggle {
      cursor: pointer;
      font-size: .7rem;
      opacity: 0.4;
      transition: all 0.3s;
      text-align: right;
      &:hover {
        opacity: 1;
      }
    }
  }

  .category,
  a.category {
    cursor: pointer;
    display: flex;
    margin: 1.5rem 0 0.65rem;
    font-weight: 500;
    color: #fff !important;
    opacity: 0.9 !important;

    align-items: center;

    svg {
      margin-right: 15px;
      opacity: 0.8;
      stroke-width: 1 !important;
    }
  }

  ul,
  li {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  a {
    display: block;
    text-decoration: none;
    opacity: 0.85;
    transition: all 0.3s;
    padding: 0.5rem 0;
    color: #cbcdd8 !important;
    position: relative;

    &:hover,
    &.active {
      opacity: 1;
      color: #fff !important;
    }
  }

  .items {
    display: none;
    margin-left: 35px;
  }

  .expanded + .items {
    display: block;
  }

  .items a {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .toggle-subcategory {
    margin-top: 3px;
    opacity: 0.4;
    transition: all 0.3s;
    text-transform: uppercase;
    letter-spacing: 1px;
    font-size: 0.7rem;
    &:hover {
      opacity: 1;
    }
  }

  .subcategory {
    display: none;
    margin: 0.5rem 1rem 1rem;

    &.expanded {
      display: block;
    }

    a {
      border-left: 2px solid #ffffff44;
      padding-left: 1.5rem;
    }

    .active {
      border-left: 2px solid #ffffff88;
    }
  }

  /* Mobile */
  @media (max-width: 800px) {
    padding: 2rem;
    grid-template-columns: 1fr;
    justify-content: flex-start;
    box-sizing: border-box;

    > div {
      width: 100%;
      box-sizing: border-box;
    }

    .toggle,
    .toggle-subcategory {
      margin-right: 0 !important;
    }
  }

  @media (max-width: 800px) {
    .category, a.category {
      margin: 1rem 0 0.65rem;
    }
    font-size: .85rem;
  }
`;

const Content = styled.div`
  > div {
    padding: 1rem 4rem 25vh;
  }

  @media (max-width: 800px) {
    > div {
      padding: 2rem;
    }
  }
`;

const Hero = styled.div`
  padding: 8vh 4rem 7vh !important;

  @media (max-width: 800px) {
    box-sizing: border-box;
    width: 100%;
    padding: 4vh 2rem 0 !important;

    h1 {
      font-size: 42px;
      padding: 0;
    }
  }
`;

const Discover = styled.div`
  > div {
    display: grid;
    grid-template-columns: 3fr 2fr;
    grid-gap: 2rem;
  }

  ul {
    margin-top: 3.5rem;
  }

  @media (max-width: 800px) {
    > div {
      grid-template-columns: 1fr;
      grid-gap: 0;
    }
    ul {
      margin-top: 1rem;
    }
  }
`;
