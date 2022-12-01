import styled from "@emotion/styled";
import Head from "next/head";
import { useRouter } from "next/router";

import Footer from "../shared/Footer";
import Header from "../shared/Header";
import Nav from "../shared/legacy/nav";
import ThemeToggleButton from "../shared/legacy/ThemeToggleButton";
import Container from "../shared/layout/Container";
import Tags from "../shared/Blog/Tags";

export default function BlogLayout(props) {
  const router = useRouter();
  const { showHidden } = router.query;

  const content = props.content.map(JSON.parse);
  const visiblePosts = showHidden
    ? content
    : content
        .filter((post) => !post.hide)
        .sort((a, z) => z.date.localeCompare(a.date));

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

      <div className="home bg-slate-1000 font-sans">
        <Header />
        <Container>
          <div>
            <h1>Blog</h1>
            <span className="divider">|</span>
            <p>{description}</p>

            {focus && (
              <a href={`/blog/${focus.slug}`}>
                <div className="post-text">
                  <h2>{focus.heading}</h2>
                  <p className="byline">
                    {focus.humanDate} <Tags tags={focus.tags} />
                  </p>
                  <p>{focus.subtitle}</p>
                </div>
                {focus.image && <img src={focus.image} />}
              </a>
            )}

            <ul>
              {rest.map((item) => (
                <li className="post--item" key={item.slug}>
                  <a href={`/blog/${item.slug}`}>
                    {item.image && <img src={item.image} />}
                    <div className="post-text">
                      <h2>{item.heading}</h2>
                      <p className="byline">
                        {item.humanDate} <Tags tags={item.tags} />
                      </p>
                      <p>{item.subtitle}</p>
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

    data.tags =
      typeof data.tags === "string"
        ? data.tags.split(",").map((t) => t.trim())
        : data.tags;

    // Disregard content, as the snippet for the blog list should be in
    // the frontmatter.  Only reply with the frontmatter as a JSON string,
    // as it has dates which cannot be serialized.
    return JSON.stringify(data);
  });

  return {
    props: {
      content,
      designVersion: "2",
      meta: {
        // TODO
        title: "Write functions, Send Events",
        description:
          "Inngest is a developer platform for building, testing and deploying code that runs in response to events or on a schedule — without spending any time on infrastructure.",
      },
    },
  };
}
