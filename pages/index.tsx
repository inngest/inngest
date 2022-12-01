import React from "react";
import Header from "../shared/Header";
import Hero from "../shared/Home/Hero";
import FitsYourWorkflow from "../shared/Home/FitsYourWorkflow";
import DevUI from "../shared/Home/DevUI";
import OutTheBox from "../shared/Home/OutTheBox";
import FullyManaged from "../shared/Home/FullyManaged";
import Roadmap from "../shared/Home/Roadmap";
import SocialCTA from "../shared/Home/SocialCTA";
import Footer from "../shared/Footer";

import Patterns from "src/shared/Home/Patterns";
import GetThingsShipped from "src/shared/Home/GetThingsShipped";

export async function getStaticProps() {
  return {
    props: {
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

export default function Home() {
  return (
    <div className="home bg-slate-1000 font-sans">
      <Header />

      <Hero />

      <FitsYourWorkflow />

      <DevUI />

      <OutTheBox />

      <Patterns />

      <FullyManaged />

      <GetThingsShipped />

      <Roadmap />

      <SocialCTA />

      <Footer />
    </div>
  );
}
