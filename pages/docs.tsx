import React from "react";
import { useRouter } from "next/router";
import styled from "@emotion/styled";

import Nav from "../shared/nav";
import { getAllDocs, Categories, Category, DocScope } from "../utils/docs";
import { categoryMeta } from "../utils/docsCategories";
import Home from "../shared/Icons/Home";

export default function DocsHome(props) {
  return (
    <DocsLayout categories={props.categories}>
      <Hero>
        <h1>Documentation</h1>
        {/* TODO: Quick start guide callouts, and graphic */}
      </Hero>

      <DocsContent>
        <div>
          <h2>What is Inngest</h2>
          <p>
            Inngest is a serverless platform for running code in real-time
            whenever things happen in your business. We subscribe to every event
            in your stack, and allow you to run workflows whenever specific
            events are received.
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
      <ContentWrapper>
        <Menu>
          <div>
            <h5>
              <span>Contents</span>
              <span
                className="toggle off"
                onClick={(e) => {
                  const span = (e.target as HTMLSpanElement);
                  const on = !span.classList.contains("off");
                  document.querySelectorAll(".category").forEach(a => {
                    on ? a.classList.remove("expanded") : a.classList.add("expanded");
                  })
                  span.classList.toggle("off");
                }}
              >
                Toggle categories
              </span>
            </h5>

            <ul>
              <li>
                <a href="/docs" className="category">
                  <Home fill="#fff" size={20} /> Home
                </a>
              </li>
              {Object.values(categories).map((c) => {
                const meta = categoryMeta[c.title.toLowerCase()] || {};

                const isCurrent = !!c.pages.find(p => p.slug === router.asPath.replace("/docs/", ""))

                return (
                  <li>
                    <span
                      className={["category", isCurrent && "expanded"].filter(Boolean).join(" ")}
                      onClick={(e) => (e.target as HTMLSpanElement).classList.toggle("expanded")}
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
          </div>
        </Menu>
        <Inner>{children}</Inner>
      </ContentWrapper>
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
    <li>
      <a
        href={"/docs/" + s.slug}
        className={s.slug === currentSlug ? "active" : ""}
      >
        {s.title}
        { children.length > 0 && (<span className="toggle-subcategory" onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          document.querySelector(`#${id}`).classList.toggle("expanded")
        }}>toggle</span>)}
      </a>
      {children.length > 0 && (
        <ul id={id} className={["subcategory", isCurrent && "expanded"].filter(Boolean).join(" ")}>
          {children.map((child) => {
            return (
              <li>
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

const ContentWrapper = styled.div`
  border-top: 1px solid #ffffff19;
  display: grid;
  grid-template-columns: 2fr 4fr;
  min-height: calc(100vh - 70px);
`;

export const DocsContent = styled.div`
  display: grid;
  max-width: 800px;
  grid-template-columns: 3fr 1fr;

  h2 {
    margin-top: 4rem;
  }

  h3 {
    margin-top: 3rem;
  }


  /* "On this page" */
  h2 + h5 {
    margin-top: 3rem;
  }

`;

export const InnerDocsContent = styled.div`
  padding-bottom: 10vh;
  h2 {
    padding-top: 2rem;
    font-size: 28px;
  }

  pre {
    margin: 3rem 0;
    padding: 1rem;
    border-radius: 3px;
  }
`

const Menu = styled.div`
  border-right: 1px solid #ffffff19;
  display: flex;
  justify-content: flex-end;
  padding: 3rem 4rem 3rem 3rem;;
  background: rgba(0, 0, 0, 0.4);
  font-size: 14px;

  > div {
    max-width: 300px;
    min-width: 300px;
  }

  h5 {
    display: flex;
    justify-content: space-between;

    .toggle {
      cursor: pointer;
      opacity: .6;
      transition: all .3s;
      margin-right: -2.5rem;
      &:hover {
        opacity: 1;
      }
    }
  }

  .category,
  a.category {
    cursor: pointer;
    display: flex;
    margin: 1.5rem 0 .65rem;
    font-weight: 500;
    color: #fff !important;
    opacity: 0.9 !important;

    align-items: center;
    line-height: 1;

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

  .toggle-subcategory {
    position: absolute;
    right: -2.5rem;
    margin-top: 3px;
    opacity: .4;
    transition: all .3s;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 1px;
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
`;

const Inner = styled.div`
  box-shadow: 0 0 40px rgba(0, 0, 0, 0.4);

  > div {
    padding: 1rem 4rem;
  }
`;

const Hero = styled.div`
  padding: 8vh 4rem 7vh !important;
  background: rgba(255, 255, 255, 0.03);
`;

const Discover = styled.div`
  > div {
    display: grid;
    grid-template-columns: 3fr 2fr;
    grid-gap: 2rem;
  }

  p,
  ul {
    font-size: 14px;
  }

  ul {
    margin-top: 3.5rem;
  }
`;
