import React from "react";
import Header from "src/shared/Home/Header";
import Hero from "src/shared/Home/Hero";
import EventDriven from "src/shared/Home/EventDriven";
import DevUI from "src/shared/Home/DevUI";
import OutTheBox from "src/shared/Home/OutTheBox";
import FullyManaged from "src/shared/Home/FullyManaged";
import Roadmap from "src/shared/Home/Roadmap";
import SocialCTA from "src/shared/Home/SocialCTA";
import Footer from "src/shared/Home/Footer";

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

      <EventDriven />

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
