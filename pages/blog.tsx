import styled from "@emotion/styled";
import Head from "next/head";
import { useRouter } from "next/router";
import Image from "next/image";

import IconCalendar from "src/shared/Icons/Calendar";
import ArrowRight from "src/shared/Icons/ArrowRight";
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
        <div
          style={{
            background: "radial-gradient(circle at center, #13123B, #08090d)",
          }}
          className="absolute w-[200vw] -translate-x-1/2 -translate-y-1/2 h-[200vw] rounded-full blur-lg opacity-90"
        ></div>

        <Header />
        <Container className="pt-8">
          <div className="flex flex-col lg:flex-row gap-2 lg:gap-4 items-start lg:items-center">
            <h2 className="font-bold text-base text-white lg:border-r border-slate-600/50 pr-4">
              Blog
            </h2>
            <p className="text-slate-200 text-sm">{description}</p>
          </div>
          <div className="pt-16">
            {focus && (
              <a
                className="relative flex flex-col-reverse lg:flex-row xl:w-4/5 bg-indigo-600 rounded-lg mb-32 group   shadow-lg"
                href={`/blog/${focus.slug}`}
              >
                <div className="absolute top-0 bottom-0 -left-[40px] -right-[40px] rounded-lg bg-indigo-500 opacity-20 rotate-1 -z-0 mx-5"></div>
                <div className="lg:w-2/5 p-8 flex flex-col items-start justify-between relative z-10">
                  <div>
                    <span className="inline-flex text-indigo-50 mb-3 text-xs font-semibold bg-indigo-700/50 px-3 py-1.5 rounded">
                      Latest Post
                    </span>
                    <h2 className="text-xl md:text-2xl lg:text-xl xl:text-2xl text-white mb-1 font-medium">
                      {focus.heading}
                    </h2>
                    <p className="text-slate-200 text-sm font-medium mb-4 flex gap-1 items-center">
                      <IconCalendar />
                      {focus.humanDate} <Tags tags={focus.tags} />
                    </p>
                    <p className="text-slate-100">{focus.subtitle}</p>
                  </div>
                  <span className="px-4 text-sm font-medium inline-flex mt-4 bg-slate-800 text-slate-50 py-1.5 rounded-full group-hover:bg-slate-700">
                    Read article
                    <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
                  </span>
                </div>
                {focus.image && (
                  <div className="lg:w-3/5 flex rounded-t-lg lg:rounded-t-none lg:rounded-r-lg relative group-hover:scale-105 group-hover:rounded-lg transition-all">
                    <Image
                      className="rounded-t-lg lg:rounded-t-none lg:rounded-r-lg group-hover:rounded-lg"
                      src={focus.image}
                      width={900}
                      height={900 / 2}
                      quality={95}
                    />
                  </div>
                )}
              </a>
            )}

            <ul className="grid grid-cols-1 md:grid-cols-2 gap-x-8 lg:gap-x-4 xl:gap-x-8 lg:grid-cols-3  gap-y-20">
              {rest.map((item) => (
                <li key={item.slug}>
                  <a
                    href={`/blog/${item.slug}`}
                    className="group flex flex-col rounded-lg ease-out transition-all "
                  >
                    {item.image && (
                      <div className="flex rounded-lg shadow group-hover:scale-105 transition-all">
                        {/* We use 720 as the responsive view goes full width at 720px viewport width */}
                        <Image
                          className="rounded-lg"
                          src={item.image}
                          width={720}
                          height={720 / 2}
                        />
                      </div>
                    )}
                    <div className="pt-4 xl:pt-6 xl:py-4">
                      <h2 className="text-base xl:text-lg text-white mb-1 group-hover:text-indigo-400 transition-all">
                        {item.heading}
                      </h2>
                      <p className="text-slate-400 text-sm font-medium mb-4 mt-2 flex items-center gap-1">
                        <IconCalendar />
                        {item.humanDate} <Tags tags={item.tags} />
                      </p>
                      <p className="text-slate-300 text-sm">{item.subtitle}</p>
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
        title: "Product & Engineering Blog",
        description: `Updates from the Inngest team about our product, engineering, and community.`,
      },
    },
  };
}
