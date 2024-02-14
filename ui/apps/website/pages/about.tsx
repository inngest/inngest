import clsx from 'clsx';
import { Button } from 'src/shared/Button';
import Footer from 'src/shared/Footer';
import Header from 'src/shared/Header';
import ArrowRight from 'src/shared/Icons/ArrowRight';
import Container from 'src/shared/layout/Container';
import classNames from 'src/utils/classNames';

const TEAM = [
  {
    name: 'Tony Holdstock-Brown',
    role: 'CEO & Founder',
    avatar: '/assets/team/tony-2022-10-18.jpg',
  },
  {
    name: 'Dan Farrelly',
    role: 'CTO & Founder',
    avatar: '/assets/team/dan-f-2023-06-26.jpg',
  },
  {
    name: 'Jack Williams',
    role: 'Founding Engineer',
    avatar: '/assets/team/jack-2022-10-10.jpg',
  },
  {
    name: 'Igor Gassmann',
    role: 'Engineer',
    avatar: '/assets/team/igor-g-2023-06-26.jpg',
  },
  {
    name: 'Aaron Harper',
    role: 'Engineer',
    avatar: '/assets/team/aaron-h-2023-06-26.jpg',
  },
  {
    name: 'Ana Filipa de Almeida',
    role: 'Engineer',
    avatar: '/assets/team/ana-a-2023-06-26.jpg',
  },
  {
    name: 'Darwin Wu',
    role: 'Engineer',
    avatar: '/assets/team/darwin-w-2023-06-26.jpg',
  },
  {
    name: 'John Bucta',
    role: 'Designer',
    avatar: '/assets/team/john-b-2024-01-21.jpg',
  },
  {
    name: 'Sylwia Vargas',
    role: 'Dev Rel',
    avatar: '/assets/team/sylwia-v-2024-01-21.jpg',
  },
];

const INVESTORS: {
  name: string;
  logo: string;
  maxWidth?: string;
  featured?: boolean;
}[] = [
  {
    name: 'GGV Capital',
    logo: '/assets/about/ggv-capital-logo-white.png',
    maxWidth: '200px',
    featured: true,
  },
  {
    name: 'Afore.vc',
    logo: '/assets/about/afore-capital-white.png',
    maxWidth: '200px',
    featured: true,
  },
  {
    name: 'Kleiner Perkins',
    logo: '/assets/about/kleiner-perkins-white.png',
  },
  {
    name: 'Banana Capital',
    logo: '/assets/about/banana-capital-white.png',
  },
  {
    name: 'Comma Capital',
    logo: '/assets/about/comma-capital-white.png',
  },
];
const ANGELS: {
  name: string;
  bio: string;
  avatar?: string;
  featured?: boolean;
}[] = [
  {
    name: 'Guillermo Rauch',
    bio: 'CEO of Vercel',
    featured: true,
    avatar: '/assets/about/guillermo-rauch-avatar.jpg',
  },
  {
    name: 'Tom Preston-Werner',
    bio: 'Founder of Github',
    featured: true,
    avatar: '/assets/about/tom-preston-werner-avatar.png',
  },
  {
    name: 'Jason Warner',
    bio: 'Former CTO at GitHub',
  },
  {
    name: 'Jake Cooper',
    bio: 'Founder at Railway',
  },
  {
    name: 'Tristan Handy',
    bio: 'CEO & Founder at dbt Labs',
  },
  {
    name: 'Oana Olteanu',
    bio: 'Partner at Signalfire',
  },
  {
    name: 'Ian Livingstone',
    bio: 'Technical Advisor at Snyk',
  },
  {
    name: 'Pim De Witte',
    bio: 'CEO at Medal.tv',
  },
];

// Used for key announcements and significant thought leadership for investors
// or potential job applicants
const FEATURED_BLOG_POSTS: { title: string; href: string }[] = [
  {
    title: 'Inngest Raises $3M Seed led by GGV Capital',
    href: '/blog/announcing-inngest-seed-financing',
  },
  {
    title: 'Inngest: Add Superpowers To Serverless Functions',
    href: '/blog/inngest-add-super-powers-to-serverless-functions',
  },
  {
    title: 'Partnership: Vercel + Inngest - The fastest way to ship background functions',
    href: '/blog/vercel-integration',
  },
  {
    title: "Completing the Jamstack: What's needed in 2022?",
    href: '/blog/completing-the-jamstack',
  },
];

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: 'About Inngest',
        description:
          'Inngest is the developer platform for easily building reliable workflows with zero infrastructure',
      },
      designVersion: '2',
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
            <header className="mt-8 text-center lg:my-24">
              <h1 className="mb-6 mt-2 bg-gradient-to-r from-[#f2e3ff] via-white to-[#e3e9ff] bg-clip-text pr-4 text-2xl font-bold tracking-tighter text-transparent drop-shadow md:text-5xl">
                About Inngest
              </h1>
              <p className="font-regular mx-auto mt-8 max-w-lg text-lg">
                Inngest is the developer platform for easily building reliable workflows with zero
                infrastructure.
              </p>
            </header>

            <div className="prose prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert mx-auto max-w-[800px] text-center text-slate-300">
              <p>
                Shipping reliable background jobs and workflows are a time consuming and frustrating
                experience for any software team. Local development is painful. Managing
                infrastructure is tedious. Days to weeks of developer time is lost doing this work
                at every company.
              </p>
              <p>
                Inngest is solving this problem for every software team, no matter team size or
                experience.
              </p>
            </div>

            <div className="mt-8 flex justify-center lg:mt-12">
              <a
                href="/blog/announcing-inngest-seed-financing"
                className="group inline-flex flex-shrink-0 items-center gap-0.5 rounded-full border border-indigo-500/50 py-2 pl-6 pr-5 font-medium text-white transition-all hover:bg-indigo-500/10"
              >
                News: Inngest Raises $3M Seed led by GGV Capital & Guillermo Rauch
                <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
              </a>
            </div>
          </div>

          <div className="mx-auto my-32 text-slate-300">
            <h2 className="text-center text-xl font-medium text-white sm:text-2xl lg:text-3xl">
              Our Team
            </h2>
            <p className="mt-2 text-center text-slate-400">
              We've built and scaled systems for years and think that developers deserve something
              better.
            </p>
            <div
              className={clsx(
                'mb-6 mt-20 grid grid-cols-2 items-center gap-x-10 gap-y-16 md:px-24 lg:grid-cols-3',
                TEAM.length % 3 === 0 && 'xl:grid-cols-3',
                TEAM.length % 2 === 0 && 'xl:grid-cols-4'
              )}
            >
              {TEAM.map((person) => {
                return (
                  <div key={person.name} className="flex flex-col items-center">
                    <img className="w-20 rounded-lg" src={person.avatar} />

                    <h3 className="mb-1 mt-4 text-base font-medium text-slate-200">
                      {person.name}
                    </h3>
                    <p className="text-sm leading-5 text-slate-400" style={{ lineHeight: '1.5em' }}>
                      {person.role}
                    </p>
                  </div>
                );
              })}
            </div>
          </div>

          <aside className="m-auto mt-32 max-w-[720px] rounded-xl border-slate-900/50 bg-slate-950/20 p-2.5">
            <div className="rounded-lg border border-slate-900 bg-slate-950 px-6 py-8 text-center shadow md:p-12 md:px-10">
              <h3 className="text-md mb-4 font-semibold text-white lg:text-lg">
                Want to join the team?
              </h3>
              <p className="mb-8 text-base text-slate-400">
                We're just getting started and are looking for people that want to contribute highly
                to an early-stage startup focused on solving developer problems.
              </p>
              <Button href="/careers?ref=about" arrow="right">
                View the open roles
              </Button>
            </div>
          </aside>
        </Container>
        <div className="bg-slate-1050/50 -mt-36 pb-32 pt-60">
          <Container className="m-auto">
            <div className="mx-auto py-6">
              <h2 className="mb-10 text-center text-xl font-medium text-white sm:text-2xl lg:text-3xl">
                Our Investors
              </h2>
            </div>
            <div className="mb-12 grid items-center gap-8 pb-6 sm:grid-cols-2 md:grid-cols-6">
              {INVESTORS.map((investor) => {
                return (
                  <div
                    className={classNames(
                      investor.featured ? 'mx-auto md:col-span-3' : 'mx-auto md:col-span-2',
                      'flex h-[130px] w-full items-center justify-center rounded border  border-slate-900 bg-slate-950 p-10 shadow'
                    )}
                  >
                    <img
                      key={investor.name}
                      style={{ maxHeight: '50px' }}
                      src={investor.logo}
                      alt={investor.name}
                    />
                  </div>
                );
              })}
            </div>

            <div className="mt-20 grid gap-4 text-center sm:grid-cols-2">
              {ANGELS.map((a, idx) =>
                a.featured ? (
                  <div key={a.name} className="mb-4 flex flex-col items-center gap-4 text-lg">
                    <img
                      src={a.avatar}
                      alt={`Image of ${a.name}`}
                      className="h-16 w-16 rounded-lg"
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
            <div className="pb-16 pt-32">
              <h2 className="mb-8 text-center text-xl font-normal sm:text-2xl">From our blog</h2>

              <div className=" flex flex-col items-center justify-center gap-4">
                {FEATURED_BLOG_POSTS.map((p, idx) => (
                  <p key={p.href} className="text-base">
                    <a className="text-indigo-400" href={`${p.href}?ref=about-page`}>
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
