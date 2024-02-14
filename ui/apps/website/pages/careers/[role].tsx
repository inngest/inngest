import Head from 'next/head';
import { MDXRemote } from 'next-mdx-remote';
import { Button } from 'src/shared/Button';
import Footer from 'src/shared/Footer';
import Header from 'src/shared/Header';
import Container from 'src/shared/layout/Container';
import { MDXContent, loadMarkdownFile, loadMarkdownFilesMetadata } from 'utils/markdown';

import { AboutInngest, Benefits, type Role } from './index';

const dir = 'careers/_roles';

export async function getStaticProps({ params }) {
  const role = await loadMarkdownFile<Role>(dir, params.role);
  return {
    props: {
      role: JSON.stringify(role),
      meta: {
        title: `Careers - ${role.metadata.title}`,
      },
      designVersion: '2',
    },
  };
}

export async function getStaticPaths() {
  const roles = await loadMarkdownFilesMetadata(dir);
  const visibleRoles = roles.filter((r) => !r.hidden);
  const paths = visibleRoles.map((role) => `/careers/${role.slug}`);
  return { paths, fallback: false };
}

const removeLinks = (md: string) => {
  return md.replace(/\((.+)\)\[.+\]/, '$1');
};

const parseJobDescription = (content: string) => {
  // Split on h3's
  const sections = content.split('###');
  const responsibilitiesSection = sections.find((s) => s.match(/^\s+What you'll do/i));
  let responsibilities = [];
  if (responsibilitiesSection) {
    const bulletPoints = responsibilitiesSection
      .replace(/^\s+What you'll do\s+/i, '')
      .split('- ')
      .filter((l) => l.length);
    responsibilities = bulletPoints.map((l) => removeLinks(l.replace(/\s+$/, '')));
  }

  return {
    description: removeLinks(sections[0]),
    responsibilities: responsibilities.join(' '),
  };
};

export default function Careers(props) {
  const role: MDXContent<Role> = JSON.parse(props.role);

  const parts = parseJobDescription(role.content);
  const structuredData = {
    '@context': 'https://schema.org',
    '@type': 'JobPosting',
    title: role.metadata.title,
    industry: 'Software',
    occupationalCategory: 'Software Engineer',
    employmentType: 'Full-time',
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
      <div className="font-sans">
        <Header />
        <Container>
          <article>
            <main className="m-auto max-w-[65ch] pt-16">
              <header className="m-auto max-w-[65ch] pt-12 lg:pt-24">
                <h1 className="mb-2 text-2xl font-medium tracking-tighter text-white md:mb-4 md:text-4xl lg:leading-loose xl:text-5xl">
                  {role.metadata.title}
                </h1>
                <p className="font-medium text-indigo-400">{role.metadata.location}</p>
              </header>
              <div className="prose prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert mx-auto my-20 text-slate-300">
                <AboutInngest heading={false} />

                <h2>The role</h2>
                <MDXRemote compiledSource={role.compiledSource} scope={{}} frontmatter={{}} />

                <Benefits />

                <h2>How to apply</h2>
                <p>
                  To apply, <a href={role.metadata.applicationURL}>complete this application</a>.
                  Please include:
                </p>
                <ul>
                  <li>Your resume</li>
                  <li>Why you'd like to join our team</li>
                  <li>
                    Links to your: Github, Linkedin, Twitter, design portfolio, etc. (if applicable)
                  </li>
                  <li>
                    If applicable (i.e. for DevRel), provide samples of your work (writing, video
                    content, conference talks, etc.)
                  </li>
                  <li>Your location</li>
                </ul>
              </div>
              <aside className="m-auto my-12 flex max-w-[65ch] flex-col items-start gap-4 rounded-lg border border-indigo-900/50 bg-indigo-900/20 px-6  py-5 leading-relaxed text-indigo-100">
                <p className="text-sm lg:text-base">Have any questions about a role?</p>
                <Button href="mailto:careers@inngest.com" arrow="right">
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
