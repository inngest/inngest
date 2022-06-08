import styled from "@emotion/styled";
import Head from "next/head";

import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";
import ThemeToggleButton from "../shared/ThemeToggleButton";
import { Wrapper } from "../shared/blog";

export default function BlogLayout(props) {
  const content = props.content.map(JSON.parse);
  const visiblePosts = content.filter((post) => !post.hide);

  const focus = visiblePosts.find((c) => c.focus);
  const rest = visiblePosts
    .filter((c) => !focus || c.slug !== focus.slug)
    .sort((a, z) => z.date.localeCompare(a.date));

  const description = `Updates from the Inngest team about our product, engineering, and community.`;

  return (
    <>
      <Head>
        <title>Inngest → Product & Engineering blog</title>
        <meta name="description" content={description}></meta>
        <meta
          property="og:title"
          content="Inngest → Product & Engineering blog"
        />
        <meta property="og:description" content={description} />
      </Head>

      <ThemeToggleButton isFloating={true} />

      <Wrapper>
        <Nav />

        <Main>
          <Header>
            <h1>Blog</h1>
            <span className="divider">|</span>
            <p>{description}</p>
          </Header>

          {focus && (
            <FocusPost href={`/blog/${focus.slug}`}>
              <div className="post-text">
                <h2>{focus.heading}</h2>
                <p className="byline">{focus.humanDate}</p>
                <p>{focus.subtitle}</p>
              </div>
              {focus.image && <img src={focus.image} />}
            </FocusPost>
          )}

          <List>
            {rest.map((item) => (
              <PreviousPost
                href={`/blog/${item.slug}`}
                className="post--item"
                key={item.slug}
              >
                {item.image && <img src={item.image} />}
                <div className="post-text">
                  <h2>{item.heading}</h2>
                  <p className="byline">{item.humanDate}</p>
                  <p>{item.subtitle}</p>
                </div>
              </PreviousPost>
            ))}
          </List>
        </Main>
        <Footer />
      </Wrapper>
    </>
  );
}

// This function also gets called at build time to generate specific content.
export async function getStaticProps() {
  // These are required here as this function is not included in frontend
  // browser builds.
  const fs = require("fs");

  // Iterate all files in the blog posts directory, then parse the markdown.
  const content = fs.readdirSync("./pages/blog/_posts/").map((fname) => {
    const matter = require("gray-matter");
    const readingTime = require("reading-time");

    const source = fs.readFileSync("./pages/blog/_posts/" + fname);

    const { data, content } = matter(source);
    data.reading = readingTime(content);
    data.humanDate = data.date.toLocaleDateString();
    data.slug = fname.replace(/.mdx?/, "");

    // Disregard content, as the snippet for the blog list should be in
    // the frontmatter.  Only reply with the frontmatter as a JSON string,
    // as it has dates which cannot be serialized.
    return JSON.stringify(data);
  });

  return { props: { content } };
}

const Main = styled.main`
  margin: 1rem auto 4rem;
  max-width: 980px;

  @media (max-width: 1000px) {
    margin-left: 1.5rem;
    margin-right: 1.5rem;
  }
`;

const Header = styled.header`
  display: flex;
  margin: 3rem auto 4rem;
  line-height: 1em;
  align-items: center;
  h1 {
    white-space: nowrap;
  }
  h1,
  p {
    font-size: 0.8rem;
    line-height: 1em;
    padding: 0;
  }
  p {
    color: var(--font-color-secondary);
  }
  .divider {
    margin: 0 0.5rem;
    font-size: 0.8rem;
    color: var(--stroke-color);
  }
`;

const BlogPost = styled.a`
  display: flex;

  flex-direction: column;
  color: var(--font-color-primary);
  text-decoration: none;

  .byline {
    margin: 0.5rem 0;
    font-size: 0.8rem;
    color: var(--font-color-secondary);
  }
  p {
    margin-top: 1.2rem;
    font-size: 0.9rem;
  }
`;

const FocusPost = styled(BlogPost)`
  width: 100%;
  margin: 3rem 0;
  align-items: center;
  flex-direction: row;

  img {
    margin-left: 1.6rem;
    max-width: 45%;
    border-radius: var(--border-radius);
  }

  @media (max-width: 800px) {
    flex-direction: column-reverse;
    img {
      margin: 0 0 1rem;
      max-width: 100%;
    }
  }
`;

const PreviousPost = styled(BlogPost)`
  align-items: start;
  /* background: var(--highlight-color); */
  border: 1px solid var(--stroke-color);
  border-radius: var(--border-radius);
  overflow: hidden;

  .post-text {
    padding: 1.2rem;
  }

  h2 {
    font-size: 1.3rem;
    line-height: 1.4em;
  }

  img {
    align-self: center;
    max-width: 100%; // a 600x300px image filles the width
    width: auto;
  }
`;

const List = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-gap: 2rem;
  margin: 5rem 2rem 20vh;

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
  }
`;
