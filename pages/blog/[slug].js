import styled from "@emotion/styled";

import Footer from "../../shared/footer";
import Nav from "../../shared/nav";
import Content from "../../shared/content";
import { Wrapper, Inner } from "../../shared/blog";

import { serialize } from 'next-mdx-remote/serialize'
import { MDXRemote } from 'next-mdx-remote'

export default function BlogLayout(props) {
  const scope = JSON.parse(props.post.scope);
  return (
    <>
      <Wrapper>
        <Nav />
        <Content>
          <Inner>
            <div>
              <h1>{scope.heading}</h1>
              <p className="blog--date">{scope.humanDate} &middot; {scope.reading.text}</p>
              <MDXRemote compiledSource={props.post.compiledSource} scope={scope} />
            </div>
          </Inner>
        </Content>
        <Footer />
      </Wrapper>
    </>
  );
}

// This function gets called at build time to figure out which URLs
// we need to statically compile.
// 
// These URLs will be treated as individual pages. getStaticProps is
// called for each URL with the slug in params.
export async function getStaticPaths() {
  const fs = require('fs');
  const paths = fs.readdirSync("./pages/blog/_posts/").map(fname => {
    return `/blog/${fname.replace(/.mdx?/, "")}`;
  });
  return { paths, fallback: false };
}


// This function also gets called at build time to generate specific content.
export async function getStaticProps({ params }) {
  // These are required here as this function is not included in frontend
  // browser builds.
  const fs = require('fs');
  const readingTime = require('reading-time');
  const matter = require('gray-matter');

  const source = fs.readFileSync("./pages/blog/_posts/" + params.slug + ".md");
  const { content, data } = matter(source)

  data.reading = readingTime(content)
  // Format the reading date.
  data.humanDate = data.date.toLocaleDateString();

  // type Post = {
  //   compiledSource: string,
  //   scope: string,
  // }
  const post = await serialize(content, { scope: JSON.stringify(data) })
  return { props: { post } };
}
