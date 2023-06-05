import React from "react";
import Header from "../shared/Header";
import Hero from "../shared/Home/Hero";
import ShipInHours from "../shared/Home/ShipInHours";
import LocalDev from "../shared/Home/LocalDev";
import OutTheBox from "../shared/Home/OutTheBox";
import FullyManaged from "../shared/Home/FullyManaged";
import Roadmap from "../shared/Home/Roadmap";
import SocialCTA from "../shared/Home/SocialCTA";
import Footer from "../shared/Footer";
import CustomerQuote from "src/shared/CustomerQuote";

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
    <div className="home font-sans">
      <Header />

      <Hero />

      <ShipInHours />

      <LocalDev />

      <CustomerQuote
        className="mt-20 md:mt-4 mb-40"
        logo="/assets/customers/ocoya.svg"
        text="At Ocoya, we were struggling with the complexities of managing our
              social media and e-commerce workflows. Thanks to Inngest, we were
              able to simplify our development process, speed up our time to
              market, and deliver a better customer experience. Inngest has
              become an essential tool in our tech stack, enabling us to focus
              on delivering a world-class product to our users."
        cta={{
          href: "/customers/ocoya?ref=homepage",
          text: "Read the case study",
        }}
      />

      <OutTheBox />

      <GetThingsShipped />

      <FullyManaged />

      <Patterns />

      {/* <Roadmap /> */}

      <SocialCTA />

      <Footer />
    </div>
  );
}
