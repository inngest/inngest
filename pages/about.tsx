import { useEffect, useState } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";

import Block from "../shared/Block";

const MISSION = "To accelerate the adoption of event-based architecture.";

const TEAM = [
  {
    name: "Tony Holdstock-Brown",
    role: "CEO & Founder",
    bio: (
      <>
        Head of Engineering at{" "}
        <span className="text-almost-black">Uniform Teeth</span>
      </>
    ),
    avatar: "/assets/team/tony-2022-10-18.jpg",
  },
  {
    name: "Dan Farrelly",
    role: "Founder",
    bio: (
      <>
        CTO at <span className="text-almost-black">Buffer</span>. Created{" "}
        <span className="text-almost-black">Timezone.io</span>.
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
    logo: "/assets/about/afore-capital-dark.png",
    maxWidth: "200px",
  },
  {
    name: "Kleiner Perkins",
    logo: "/assets/about/kleiner-perkins-dark.png",
  },
  {
    name: "Banana Capital",
    logo: "/assets/about/banana-capital-dark.png",
  },
  {
    name: "Comma Capital",
    logo: "/assets/about/comma-capital-dark.png",
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

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "About Us",
        description: MISSION,
      },
    },
  };
}

export default function Home() {
  return (
    <div>
      <Nav />

      <div>
        {/* Content layout */}
        <div className="mx-auto my-12 px-10 lg:px-16 max-w-3xl">
          <header className="lg:my-24 mt-8">
            <span className="text-sm font-bold uppercase gradient-text-ltr">
              Our Mission
            </span>
            <h1 className="mt-2 mb-6 pr-4 text-2xl md:text-4xl leading-tight">
              {MISSION}
            </h1>
            <p>
              We believe that event-based systems can be beautifully simple and
              we're building the platform to enable developers to build amazing
              products without the overhead.
            </p>
          </header>
        </div>
      </div>

      <div
        style={{ backgroundColor: "#f8f7fa" }}
        className="background-grid-texture"
      >
        <div className="container mx-auto px-10 lg:px-16 max-w-3xl py-8">
          <div className="mx-auto my-6">
            <h2 className="text-xl sm:text-2xl font-normal">Our Team</h2>
            <p className="my-2">
              We've built and scaled event-based architectures for years and
              think that developers deserve something better. Interested?{" "}
              <a href="mailto:founders@inngest.com">Drop us a line</a>.
            </p>
          </div>
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
                    <span className="text-slate-500">
                      {person.bio && "Past: "}
                      {person.bio}
                    </span>
                  </p>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div>
        <div className="container mx-auto px-10 lg:px-16 max-w-3xl py-8">
          <div className="mx-auto py-6">
            <h2 className="text-xl sm:text-2xl font-normal">Our Investors</h2>
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
        </div>
      </div>

      <div style={{ marginTop: 100 }}>
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
