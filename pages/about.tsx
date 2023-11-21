import classNames from "src/utils/classNames";

import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import ArrowRight from "src/shared/Icons/ArrowRight";
import { Button } from "src/shared/Button";

const TEAM = [
  {
    name: "Tony Holdstock-Brown",
    role: "CEO & Founder",
    avatar: "/assets/team/tony-2022-10-18.jpg",
  },
  {
    name: "Dan Farrelly",
    role: "CTO & Founder",
    avatar: "/assets/team/dan-f-2023-06-26.jpg",
  },
  {
    name: "Jack Williams",
    role: "Founding Engineer",
    avatar: "/assets/team/jack-2022-10-10.jpg",
  },
  {
    name: "Igor Gassmann",
    role: "Engineer",
    avatar: "/assets/team/igor-g-2023-06-26.jpg",
  },
  {
    name: "Aaron Harper",
    role: "Engineer",
    avatar: "/assets/team/aaron-h-2023-06-26.jpg",
  },
  {
    name: "Ana Filipa de Almeida",
    role: "Engineer",
    avatar: "/assets/team/ana-a-2023-06-26.jpg",
  },
  {
    name: "Darwin Wu",
    role: "Engineer",
    avatar: "/assets/team/darwin-w-2023-06-26.jpg",
  },
];

const INVESTORS: {
  name: string;
  logo: string;
  maxWidth?: string;
  featured?: boolean;
}[] = [
  {
    name: "GGV Capital",
    logo: "/assets/about/ggv-capital-logo-white.png",
    maxWidth: "200px",
    featured: true,
  },
  {
    name: "Afore.vc",
    logo: "/assets/about/afore-capital-white.png",
    maxWidth: "200px",
    featured: true,
  },
  {
    name: "Kleiner Perkins",
    logo: "/assets/about/kleiner-perkins-white.png",
  },
  {
    name: "Banana Capital",
    logo: "/assets/about/banana-capital-white.png",
  },
  {
    name: "Comma Capital",
    logo: "/assets/about/comma-capital-white.png",
  },
];
const ANGELS: {
  name: string;
  bio: string;
  avatar?: string;
  featured?: boolean;
}[] = [
  {
    name: "Guillermo Rauch",
    bio: "CEO of Vercel",
    featured: true,
    avatar: "/assets/about/guillermo-rauch-avatar.jpg",
  },
  {
    name: "Tom Preston-Werner",
    bio: "Founder of Github",
    featured: true,
    avatar: "/assets/about/tom-preston-werner-avatar.png",
  },
  {
    name: "Jason Warner",
    bio: "Former CTO at GitHub",
  },
  {
    name: "Jake Cooper",
    bio: "Founder at Railway",
  },
  {
    name: "Tristan Handy",
    bio: "CEO & Founder at dbt Labs",
  },
  {
    name: "Oana Olteanu",
    bio: "Partner at Signalfire",
  },
  {
    name: "Ian Livingstone",
    bio: "Technical Advisor at Snyk",
  },
  {
    name: "Pim De Witte",
    bio: "CEO at Medal.tv",
  },
];

// Used for key announcements and significant thought leadership for investors
// or potential job applicants
const FEATURED_BLOG_POSTS: { title: string; href: string }[] = [
  {
    title: "Inngest Raises $3M Seed led by GGV Capital",
    href: "/blog/announcing-inngest-seed-financing",
  },
  {
    title: "Inngest: Add Superpowers To Serverless Functions",
    href: "/blog/inngest-add-super-powers-to-serverless-functions",
  },
  {
    title:
      "Partnership: Vercel + Inngest - The fastest way to ship background functions",
    href: "/blog/vercel-integration",
  },
  {
    title: "Completing the Jamstack: What's needed in 2022?",
    href: "/blog/completing-the-jamstack",
  },
];

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "About Inngest",
        description:
          "Inngest is the developer platform for easily building reliable workflows with zero infrastructure",
      },
      designVersion: "2",
    },
  };
}

export default function About() {
  return (
    <div className="font-sans">
      <Header />
      <main className="pt-16">
        <Container className="m-auto">
          <div className="mx-auto max-w-4xl">
            <header className="lg:my-24 mt-8 text-center">
              <h1 className="mt-2 mb-6 pr-4 text-2xl md:text-5xl tracking-tighter font-bold bg-clip-text text-transparent bg-gradient-to-r from-[#E2BEFF] via-white to-[#AFC1FF] drop-shadow">
                Ship More Reliable Workflows. Faster.
              </h1>
              <p className="mt-8 mx-auto max-w-lg text-lg font-regular">
                Inngest is the developer platform for easily building reliable
                workflows with zero infrastructure.
              </p>
            </header>

            <div className="mx-auto max-w-[800px] text-center prose text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert">
              <p>
                Shipping reliable background jobs and workflows are a time
                consuming and frustrating experience for any software team.
                Local development is painful. Managing infrastructure is
                tedious. Days to weeks of developer time is lost doing this work
                at every company.
              </p>
              <p>
                Inngest is solving this problem for every software team, no
                matter team size or experience.
              </p>
            </div>

            <div className="mt-8 lg:mt-12 flex justify-center">
              <a
                href="/blog/announcing-inngest-seed-financing"
                className="group inline-flex gap-0.5 items-center rounded-full font-medium pl-6 pr-5 py-2 border border-indigo-500/50 hover:bg-indigo-500/10 transition-all text-white flex-shrink-0"
              >
                News: Inngest Raises $3M Seed led by GGV Capital & Guillermo
                Rauch
                <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
              </a>
            </div>
          </div>

          <div className="my-32 mx-auto text-slate-300">
            <h2 className="text-xl sm:text-2xl lg:text-3xl font-medium text-center text-white">
              Our Team
            </h2>
            <p className="mt-2 text-center text-slate-400">
              We've built and scaled systems for years and think that developers
              deserve something better.
            </p>
            <div className="mt-20 mb-6 grid md:px-24 grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-x-10 gap-y-16 items-center">
              {TEAM.map((person) => {
                return (
                  <div key={person.name} className="flex flex-col items-center">
                    <img className="w-20 rounded-lg" src={person.avatar} />

                    <h3 className="mt-4 mb-1 text-base text-slate-200 font-medium">
                      {person.name}
                    </h3>
                    <p
                      className="text-sm leading-5 text-slate-400"
                      style={{ lineHeight: "1.5em" }}
                    >
                      {person.role}
                    </p>
                  </div>
                );
              })}
            </div>
          </div>

          <aside className="max-w-[720px] m-auto p-2.5 rounded-xl bg-slate-950/20 mt-32 border-slate-900/50">
            <div className="bg-slate-950 rounded-lg px-6 py-8 md:px-10 md:p-12 border border-slate-900 text-center shadow">
              <h3 className="text-md lg:text-lg font-semibold text-white mb-4">
                Want to join the team?
              </h3>
              <p className="text-base text-slate-400 mb-8">
                We're just getting started and are looking for people that want
                to contribute highly to an early-stage startup focused on
                solving developer problems.
              </p>
              <Button href="/careers?ref=about" arrow="right">
                View the open roles
              </Button>
            </div>
          </aside>
        </Container>
        <div className="bg-slate-1050/50 pt-60 pb-32 -mt-36">
          <Container className="m-auto">
            <div className="mx-auto py-6">
              <h2 className="text-xl sm:text-2xl lg:text-3xl font-medium text-center text-white mb-10">
                Our Investors
              </h2>
            </div>
            <div className="pb-6 grid sm:grid-cols-2 md:grid-cols-6 gap-8 mb-12 items-center">
              {INVESTORS.map((investor) => {
                return (
                  <div
                    className={classNames(
                      investor.featured
                        ? "md:col-span-3 mx-auto"
                        : "md:col-span-2 mx-auto",
                      "flex items-center bg-slate-950 rounded w-full justify-center p-10  border border-slate-900 shadow h-[130px]"
                    )}
                  >
                    <img
                      key={investor.name}
                      style={{ maxHeight: "50px" }}
                      src={investor.logo}
                      alt={investor.name}
                    />
                  </div>
                );
              })}
            </div>

            <div className="grid sm:grid-cols-2 gap-4 mt-20 text-center">
              {ANGELS.map((a, idx) =>
                a.featured ? (
                  <div
                    key={a.name}
                    className="mb-4 flex flex-col items-center gap-4 text-lg"
                  >
                    <img
                      src={a.avatar}
                      alt={`Image of ${a.name}`}
                      className="rounded-lg h-16 w-16"
                    />
                    <span>
                      {a.name}
                      <br />
                      <span className="text-slate-500">{a.bio}</span>
                    </span>
                  </div>
                ) : (
                  <div key={a.name} className="text-sm">
                    <h4>{a.name}</h4>
                    <p className="text-slate-500">{a.bio}</p>
                    <br />
                  </div>
                )
              )}
            </div>
          </Container>
        </div>
        <Container>
          {FEATURED_BLOG_POSTS.length && (
            <div className="pt-32 pb-16">
              <h2 className="text-xl sm:text-2xl font-normal text-center mb-8">
                From our blog
              </h2>

              <div className=" flex flex-col gap-4 justify-center items-center">
                {FEATURED_BLOG_POSTS.map((p, idx) => (
                  <p key={p.href} className="text-base">
                    <a
                      className="text-indigo-400"
                      href={`${p.href}?ref=about-page`}
                    >
                      {p.title} →
                    </a>
                  </p>
                ))}
              </div>
            </div>
          )}
        </Container>
      </main>

      <div className="mt-48">
        <Footer />
      </div>
    </div>
  );
}
