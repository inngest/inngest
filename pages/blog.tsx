import styled from "@emotion/styled";
import Head from "next/head";
import { useRouter } from "next/router";

import Footer from "../shared/Footer";
import Header from "../shared/Header";
import Nav from "../shared/legacy/nav";
import ThemeToggleButton from "../shared/legacy/ThemeToggleButton";
import Container from "../shared/layout/Container";
import Tags from "../shared/Blog/Tags";
import SectionHeader from "src/shared/SectionHeader";

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

      <div className="bg-slate-1000 font-sans">
        <Header />
        <Container className="pt-32">
          <SectionHeader title="Blog" lede={description} />
          <div className="pt-32">
            {focus && (
              <a
                className="flex w-4/5 bg-slate-800/30 rounded-lg mb-32"
                href={`/blog/${focus.slug}`}
              >
                <div className="w-2/5 p-8">
                  <h2 className="text-2xl text-white mb-1">{focus.heading}</h2>
                  <p className="text-slate-400 text-sm font-medium mb-4">
                    {focus.humanDate} <Tags tags={focus.tags} />
                  </p>
                  <p className="text-slate-300">{focus.subtitle}</p>
                </div>
                {focus.image && (
                  <img src={focus.image} className="w-3/5 rounded-r-lg" />
                )}
              </a>
            )}

            <ul className="grid grid-cols-3 gap-x-4 gap-y-12">
              {rest.map((item) => (
                <li key={item.slug}>
                  <a href={`/blog/${item.slug}`}>
                    {item.image && <img src={item.image} />}
                    <div className="px-4 py-6">
                      <h2 className="text-xl text-white mb-1">
                        {item.heading}
                      </h2>
                      <p className="text-slate-400 text-sm font-medium mb-4">
                        {item.humanDate} <Tags tags={item.tags} />
                      </p>
                      <p className="text-slate-300">{item.subtitle}</p>
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
