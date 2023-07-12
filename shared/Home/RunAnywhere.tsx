import Link from "next/link";
import Container from "../layout/Container";

import Heading from "./Heading";
import CustomerQuote from "./CustomerQuote";

const highlights = [
  {
    title: "Serverless, Servers or Edge",
    description:
      "Inngest functions run anywhere that you deploy your code. Mix and match for your needs, from GPU optimized VMs to instantly scaling serverless platforms.",
    img: "/assets/homepage/paths-graphic.svg",
  },
  {
    title: "Logging & observability built-in",
    description:
      "Debug issues quickly without having to leave the Inngest dashboard.",
    img: "/assets/homepage/observability-graphic.svg",
  },
  {
    title: "We call you",
    description:
      "Inngest invokes your code via HTTP at exactly the right time, injecting function state on each call.  Ship complex workflows by writing code.",
    img: "/assets/homepage/we-call-you-graphic.svg",
  },
];

export default function RunAnywhere() {
  return (
    <Container className="mt-40 lg:mt-64 mb-24 tracking-tight">
      <Heading
        title={
          <>
            Run anywhere, zero infrastructure
            <br className="hidden lg:block" /> or config required
          </>
        }
        lede="Inngest calls your code wherever it's hosted. Deploy to your existing setup, and deliver products faster without managing infrastructure."
        variant="light"
        className="mx-auto max-w-3xl text-center"
      />

      <div className="mt-8 mb-24 lg:my-24 mx-auto max-w-6xl grid md:grid-cols-3 gap-7">
        {highlights.map(({ title, description, img }, idx) => (
          <div
            key={idx}
            className="flex flex-col justify-between rounded-lg bg-gradient-to-b from-transparent to-[#D8E2FA]"
          >
            <div className="my-6 mx-9">
              <h3 className="text-xl text-slate-600 font-semibold">{title}</h3>
              <p className="my-1.5 text-sm text-slate-500 font-medium">
                {description}
              </p>
            </div>
            <img
              src={img}
              className="w-full pointer-events-none"
              alt={`Graphic for ${title}`}
            />
          </div>
        ))}
      </div>
    </Container>
  );
}
