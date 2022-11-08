import Head from "next/head";
import dynamic from "next/dynamic";
import ReactMarkdown from "react-markdown";
import Footer from "../../shared/Footer";
import Nav from "../../shared/nav";
import Content from "../../shared/content";
import Tag from "../../shared/tag";
import { Wrapper } from "../../shared/blog";
import library from "../../public/json/library.json";
import { Inner, WorkflowContent, Description } from "../../shared/libraryitem";
import { titleCase, slugify } from "../../shared/util";
const Workflow = dynamic(() => import("../../shared/Workflow/Viewer"));

const path = "$1";

//
//
// WARNING: THIS FILE IS AUTO-GENERATED VIA generate-library.js AND THE LIBRARY JSON FILE.
//
// To edit this file, change the template in `layouts/library-item.js` then run
// `make library` to regenerate the library and templates.
//
//

export default function LibraryItem() {
  const item = library.find((l) => slugify(l.title) === path);
  if (!item) {
    return null;
  }

  return (
    <>
      <Head>
        <title>{item.title} → Inngest Serverless Library</title>
      </Head>
      <Wrapper>
        <Nav />
        <Content>
          <Inner>
            <a href="/library" className="back">
              &lsaquo; Back to the library
            </a>
            <h2>{item.title}</h2>
            <p>{item.subtitle}</p>
            {item.tags.map((t) => (
              <Tag key={t}>{titleCase(t)}</Tag>
            ))}

            <WorkflowContent>
              <Workflow config={item.workflow} />
            </WorkflowContent>

            <Description>
              <ReactMarkdown children={item.description} linkTarget="_blank" />
            </Description>
          </Inner>
        </Content>
        <Footer />
      </Wrapper>
    </>
  );
}
