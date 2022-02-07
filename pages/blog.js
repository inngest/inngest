import styled from "@emotion/styled";
import Head from "next/head";

import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";
import { Wrapper } from "../shared/blog";

export default function BlogLayout(props) {
  const content = props.content.map(JSON.parse);

  const focus = content.find((c) => c.focus);
  const rest = content.filter((c) => !focus || c.slug !== focus.slug);

  return (
    <>
      <Head>
        <title>Inngest → Product & engineering blog</title>
      </Head>

      <Wrapper>
        <Nav />
        <Intro>
          <Content>
            <header>
              <h3>Inngest Blog</h3>
              <p>
                The latest news and announcements about Inngest, our ecosystem,
                product uses, and the engineering effort behind it.
              </p>
            </header>
          </Content>
        </Intro>
        <Content>
          {focus && (
            <Focus>
              <a href={`/blog/${focus.slug}`}>
                <div>
                  <h2>{focus.heading}</h2>
                  <Date>{focus.humanDate}</Date>
                  <p>{focus.subtitle}</p>
                </div>
                {focus.img && <div className="img"><img src={focus.img} /></div>}
              </a>
            </Focus>
          )}

          <List>
            {rest.map((item) => (
              <a
                href={`/blog/${item.slug}`}
                className="post--item"
                key={item.slug}
              >
                <h2>{item.heading}</h2>
                <Date>{item.humanDate}</Date>
                <p>{item.subtitle}</p>
              </a>
            ))}

            <div>
              <h2>More to come...</h2>
              <p>
                We'll be posting engineering articles, product releases and case
                studies consistently.
              </p>
            </div>
          </List>
        </Content>
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
    console.log(data);
    data.slug = fname.replace(/.mdx?/, "");

    // Disregard content, as the snippet for the blog list should be in
    // the frontmatter.  Only reply with the frontmatter as a JSON string,
    // as it has dates which cannot be serialized.
    return JSON.stringify(data);
  });

  return { props: { content } };
}

const Intro = styled.div`
  padding: 6vh 18px 15vh;
  z-index: 0;
  background: rgba(255, 255, 255, 0.03);

  header {
    max-width: 100% !important;
    padding: 0 2rem !important;
    h1 {
      opacity: 0.8;
    }
  }
`;

const Focus = styled.div`
  margin: -10vh 0 8vh;
  border: 1px solid #ffffff19;
  z-index: 1;
  background: var(--bg-dark);
  box-shadow: 0 20px 80px rgba(0, 0, 0, 0.5);
  border-radius: 3px;

  a {
    text-decoration: none;
    display: grid;
    grid-template-columns: 3fr 2fr;
    align-items: center;

    img {
      max-width: 100%;
    }

    /* Text */
    > div:first-of-type {
      padding: 4rem 3rem;
    }
  }
  h2 {
    margin: 0 0 10px;
  }

  .img { 
    display: flex;
    justify-content: center;
  }
`;

const Date = styled.div`
  font-size: 0.9rem;
  opacity: 0.7;
  margin-bottom: 1.5rem;
`;

const List = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  grid-gap: 2rem;
  padding: 0 0 20vh 0;

  > div,
  > a {
    border: 1px solid #ffffff19;
    padding: 3rem 3rem 2rem;
    text-decoration: none;
    border-radius: 3px;
  }

  h2 {
    font-size: 1.65rem;
    margin: 0 0 10px;
  }

  > div:last-of-type {
    opacity: 0.6;
  }
`;
