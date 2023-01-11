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
        title: "Reliable serverless background functions on any platform",
        description:
          "Inngest is an open source platform that enables developers to build amazing products by ensuring serverless functions are reliable, schedulable and event-driven.",
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
