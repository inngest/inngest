import styled from "@emotion/styled";
// import Footer from "../shared/legacy/Footer";
import Nav from "../shared/legacy/nav";

import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import Block from "../shared/legacy/Block";
import { Button } from "src/shared/Button";

const MISSION = "To accelerate the adoption of event-based architecture.";

const TEAM = [
  {
    name: "Tony Holdstock-Brown",
    role: "CEO & Founder",
    bio: (
      <>
        Head of Engineering at{" "}
        <span className="text-slate-100">Uniform Teeth</span>
      </>
    ),
    avatar: "/assets/team/tony-2022-10-18.jpg",
  },
  {
    name: "Dan Farrelly",
    role: "Founder",
    bio: (
      <>
        CTO at <span className="text-slate-100">Buffer</span>. Created{" "}
        <span className="text-slate-100">Timezone.io</span>.
      </>
    ),
    avatar: "/assets/team/dan-f-2022-02-18.jpg",
  },
  {
    name: "Jack Williams",
    role: "Founding Engineer",
    bio: "",
    avatar: "/assets/team/jack-2022-10-10.jpg",
  },
];

const INVESTORS = [
  {
    name: "Afore.vc",
    logo: "/assets/about/afore-capital-white.png",
    maxWidth: "200px",
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
const ANGELS = [
  {
    name: "Jason Warner",
    bio: "Former CTO @ GitHub",
  },
  {
    name: "Jake Cooper",
    bio: "Founder @ Railway",
  },
  {
    name: "Oana Olteanu",
    bio: "Partner @ Signalfire",
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
    title: "Completing the Jamstack: What's needed in 2022?",
    href: "/blog/completing-the-jamstack",
  },
  {
    title: "Modern serverless job schedulers",
    href: "/blog/modern-serverless-job-scheduler",
  },
  {
    title:
      "Open sourcing Inngest: The serverless event-driven platform for developers",
    href: "/blog/open-source-event-driven-queue",
  },
];

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "About Us",
        description: MISSION,
      },
      designVersion: "2",
    },
  };
}

export default function About() {
  return (
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
            <header className="lg:my-24 mt-8">
              <span className="text-sm font-medium uppercase gradient-text-ltr">
                Our Mission
              </span>
              <h1 className="mt-2 mb-6 pr-4 text-2xl md:text-5xl text-white font-medium tracking-tighter">
                {MISSION}
              </h1>
            </header>

            <div className="mx-auto prose text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert">
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
                products and building these systems is <em>extremely hard</em>.
              </p>
              <p>
                We believe that event-based systems can be beautifully simple
                and we're building the world's first developer platform that
                allows people to build event-driven products in minutes. Our aim
                is to give developers the superpowers they need to just build.
                Developers deserve first class local tooling and a{" "}
                <em>platform</em> that gives them everything they need to
                deliver, not just the underlying <em>plumbing</em> or
                infrastructure.
              </p>
              <p>
                We're beginning our product journey focused on the early adopter
                - the person who embraces <em>the developer cloud:</em> modern
                solutions that put developer experience at the forefront of the
                product. Our initial goal is to build the absolute best platform
                and tooling for devs to build anything that runs in the
                background using events. We're{" "}
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
            </div>

            <div className="my-16 mx-auto max-w-prose text-slate-300">
              <h2 className="text-xl sm:text-2xl font-normal">Our Team</h2>
              <p className="my-6">
                We've built and scaled event-based architectures for years and
                think that developers deserve something better.
              </p>
              <div className="mt-8 mb-6 grid sm:grid-cols-2 md:grid-cols-3 gap-10 items-start">
                {TEAM.map((person) => {
                  return (
                    <div key={person.name} className="flex flex-col">
                      <img className="w-20 rounded-lg" src={person.avatar} />
                      <h3 className="mt-4 mb-3 text-base font-normal">
                        {person.name}
                      </h3>
                      <p
                        className="text-sm leading-5"
                        style={{ lineHeight: "1.5em" }}
                      >
                        {person.role}
                        <br />
                        <span className="text-slate-400">
                          {person.bio && "Past: "}
                          {person.bio}
                        </span>
                      </p>
                    </div>
                  );
                })}
              </div>

              <aside className=" max-w-[65ch] m-auto bg-slate-900/20 text-indigo-100 flex flex-col items-start gap-4 leading-relaxed rounded-lg py-5 px-6  my-12 border border-indigo-900/50">
                <h3 className="text-md lg:text-lg">Want to join the team?</h3>
                <p className="text-sm lg:text-base">
                  Inngest is hiring for several positions across engineering and
                  DevRel. We're just getting started and are looking for people
                  that want to contribute highly to an early-stage startup
                  focused on solving developer problems.
                </p>
                <Button href="/careers?ref=about" arrow="right">
                  View the open roles
                </Button>
              </aside>

              <div className="mx-auto py-6">
                <h2 className="text-xl sm:text-2xl font-normal">
                  Our Investors
                </h2>
              </div>
              <div className="pb-6 grid sm:grid-cols-2 md:grid-cols-4 gap-10 items-center">
                {INVESTORS.map((investor) => {
                  return (
                    <img
                      key={investor.name}
                      style={{ maxHeight: "50px" }}
                      src={investor.logo}
                      alt={investor.name}
                    />
                  );
                })}
              </div>
              <div className="my-8">
                <div className="grid sm:grid-cols-2 gap-2">
                  {ANGELS.map((a, idx) => (
                    <div key={a.name} className="text-sm">
                      {a.name} / <span className="text-slate-500">{a.bio}</span>
                      <br />
                    </div>
                  ))}
                </div>
              </div>

              {FEATURED_BLOG_POSTS.length && (
                <>
                  <div className="mx-auto py-6">
                    <h2 className="text-xl sm:text-2xl font-normal">
                      From our blog
                    </h2>
                  </div>
                  <div className="">
                    {FEATURED_BLOG_POSTS.map((p, idx) => (
                      <div className="mb-2">
                        <p key={p.href} className="text-base">
                          <a href={`${p.href}?ref=about-page`}>→ {p.title}</a>
                        </p>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </div>
          </main>
        </article>
      </Container>

      <div className="mt-48">
        <Footer />
      </div>
    </div>
  );
}

const Avatar = styled.img`
  border-radius: 50%;
  width: 5rem;
  height: 5rem;
  margin-bottom: 1rem;
`;

const InvestorBlock = styled(Block)`
  display: flex;
  align-items: center;
  justify-content: center;
`;

const Grid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  grid-gap: 2rem 2rem;
  margin: 2rem 0;

  @media (max-width: 980px) {
    grid-template-columns: 1fr;
  }
`;

const Label = styled.p`
  font-size: 0.7rem;
  text-transform: uppercase;
  margin: 0.5rem 0;
  font-family: var(--font-mono);
`;

const Hero = styled.div`
  margin: 4rem 0;

  h1 {
    font-size: 2rem;
  }
  p {
    max-width: 36rem;
  }
`;
