import Head from "next/head";
import { MDXRemote } from "next-mdx-remote";

import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import Button from "src/shared/Button";
import {
  loadMarkdownFile,
  loadMarkdownFilesMetadata,
  MDXContent,
} from "utils/markdown";
import type { Role } from "./index";

const dir = "careers/_roles";

export async function getStaticProps({ params }) {
  const role = await loadMarkdownFile<Role>(dir, params.role);
  return {
    props: {
      role: JSON.stringify(role),
      meta: {
        title: `Careers - ${role.metadata.title}`,
      },
      designVersion: "2",
    },
  };
}

export async function getStaticPaths() {
  const roles = await loadMarkdownFilesMetadata(dir);
  const paths = roles.map((role) => `/careers/${role.slug}`);
  return { paths, fallback: false };
}

const removeLinks = (md: string) => {
  return md.replace(/\((.+)\)\[.+\]/, "$1");
};

const parseJobDescription = (content: string) => {
  // Split on h3's
  const sections = content.split("###");
  const responsibilitiesSection = sections.find((s) =>
    s.match(/^\s+What you'll do/i)
  );
  let responsibilities = [];
  if (responsibilitiesSection) {
    const bulletPoints = responsibilitiesSection
      .replace(/^\s+What you'll do\s+/i, "")
      .split("- ")
      .filter((l) => l.length);
    responsibilities = bulletPoints.map((l) =>
      removeLinks(l.replace(/\s+$/, ""))
    );
  }

  return {
    description: removeLinks(sections[0]),
    responsibilities: responsibilities.join(" "),
  };
};

export default function Careers(props) {
  const role: MDXContent<Role> = JSON.parse(props.role);

  const parts = parseJobDescription(role.content);
  const structuredData = {
    "@context": "https://schema.org",
    "@type": "JobPosting",
    title: role.metadata.title,
    industry: "Software",
    occupationalCategory: "Software Engineer",
    employmentType: "Full-time",
    description: parts.description,
    responsibilities: parts.responsibilities,
    datePosted: role.metadata.date,
  };

  return (
    <>
      <Head>
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify(structuredData),
          }}
        ></script>
      </Head>
      <div className="bg-slate-1000 font-sans">
        <div
          style={{
            background: "radial-gradient(circle at center, #13123B, #08090d)",
          }}
          className="absolute w-[200vw] -translate-x-1/2 -translate-y-1/2 h-[200vw] rounded-full blur-lg opacity-90"
        ></div>
        <Header />
        <Container>
          <article>
            <main className="m-auto max-w-3xl pt-16">
              <header className="pt-12 lg:pt-24 max-w-[65ch] m-auto">
                <h1 className="text-white font-medium text-2xl md:text-4xl xl:text-5xl mb-2 md:mb-4 tracking-tighter lg:leading-loose">
                  {role.metadata.title}
                </h1>
                <p className="font-medium text-indigo-400">
                  {role.metadata.location}
                </p>
              </header>
              <div className="my-20 mx-auto prose prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert">
                <p>
                  Inngest is an{" "}
                  <a href="https://github.com/inngest/inngest">open source</a>{" "}
                  platform that enables developers to build amazing products by
                  ensuring serverless functions are reliable, schedulable and
                  event-driven.
                </p>
                <p>
                  Two trends have shaped our vision for the Inngest platform:
                  event-driven systems are driving some of the world's greatest
                  products and building these systems is <em>extremely hard</em>
                  .
                </p>
                <p>
                  We believe that event-based systems can be beautifully simple
                  and we're building the world's first developer platform that
                  allows people to build event-driven products in minutes. Our
                  aim is to give developers the superpowers they need to just
                  build. Developers deserve first class local tooling and a{" "}
                  <em>platform</em> that gives them everything they need to
                  deliver, not just the underlying <em>plumbing</em> or
                  infrastructure.
                </p>
                <p>
                  We're beginning our product journey focused on the early
                  adopter - the person who embraces{" "}
                  <em>the developer cloud:</em> modern solutions that put
                  developer experience at the forefront of the product. Our
                  initial goal is to build the absolute best platform and
                  tooling for devs to build anything that runs in the background
                  using events. We're{" "}
                  <a href="https://www.inngest.com/blog/vercel-integration">
                    partnering with key companies
                  </a>{" "}
                  to fill a{" "}
                  <a href="https://www.inngest.com/blog/completing-the-jamstack">
                    key gap in the current ecosystem
                  </a>{" "}
                  and bring Inngest to the masses. We have very big plans beyond
                  that - if you're curious, drop us a note.
                </p>
                <h2>The role</h2>
                <MDXRemote compiledSource={role.compiledSource} />
                <h2>What we offer</h2>
                <ul>
                  <li>Competitive salary and equity</li>
                  <li>Remote-first - work from anywhere</li>
                  <li>Health, dental, and vision insurance (US)</li>
                  <li>
                    International employment and payroll via{" "}
                    <a href="https://www.oysterhr.com/" target="_blank">
                      OysterHR
                    </a>
                  </li>
                  <li>M2 Macbook Pro</li>
                  <li>4 weeks vacation + local national holidays</li>
                  <li>401k (US)</li>
                </ul>
                <h2>How to apply</h2>
                <p>
                  To apply, send an email to{" "}
                  <a href="mailto:careers@inngest.com">careers@inngest.com</a>.
                  Please include:
                </p>
                <ul>
                  <li>Your resume</li>
                  <li>Why you'd like to join our team</li>
                  <li>
                    Links to your: Github, Linkedin, Twitter (if applicable)
                  </li>
                  <li>
                    If applicable (i.e. for DevRel), provide samples of your
                    work (writing, video content, conference talks, etc.)
                  </li>
                  <li>Your location</li>
                </ul>
              </div>
              <aside className="max-w-[65ch] m-auto bg-indigo-900/20 text-indigo-100 flex flex-col items-start gap-4 leading-relaxed rounded-lg py-5 px-6  my-12 border border-indigo-900/50">
                <p className="text-sm lg:text-base">
                  Have any questions about a role?
                </p>
                <Button href="mailto:careers@inngest.com" arrow>
                  Email us
                </Button>
              </aside>
            </main>
          </article>
        </Container>
        <Footer />
      </div>
    </>
  );
}
