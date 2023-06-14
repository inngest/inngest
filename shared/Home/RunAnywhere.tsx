import Link from "next/link";
import Container from "../layout/Container";

import Heading from "./Heading";
import CustomerQuote from "./CustomerQuote";

const highlights = [
  {
    title: "Serverless, Servers or Edge",
    description:
      "Inngest functions can run anywhere that you deploy your code. Mix and match for your needs, from GPU optimized VMs to instantly scaling serverless platforms.",
    img: "/assets/homepage/paths-graphic.svg",
  },
  {
    title: "Logging & Observability Built-in",
    description:
      "Debug issues quickly without having to leave the Inngest dashboard.",
    img: "/assets/homepage/observability-graphic.svg",
  },
  {
    title: "We Call You",
    description:
      "Inngest invokes your code via HTTP exactly when it needs to and manages the state of your function along.",
    img: "/assets/homepage/we-call-you-graphic.svg",
  },
];

export default function RunAnywhere() {
  return (
    <Container className="mt-64 mb-24 tracking-tight">
      <Heading
        title={
          <>
            Run Anywhere, Zero Infrastructure,
            <br className="hidden lg:block" /> or Config Required
          </>
        }
        lede="Inngest invokes your background functions and workflows wherever you currently host your code. Deliver products faster and skip the hassle of managing infrastructure."
        variant="light"
        className="mx-auto max-w-3xl text-center"
      />

      <div className="my-24 mx-auto max-w-6xl grid md:grid-cols-3 gap-7">
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

      <div className="flex items-center justify-center">
        <Link
          href="/product/how-inngest-works?ref=homepage-run-anywhere"
          className="rounded-md px-3 py-1.5 text-sm font-medium bg-white transition-all text-slate-600 hover:text-slate-800 border border-slate-200 hover:bg-slate-50 whitespace-nowrap drop-shadow"
        >
          Learn How Inngest Works
        </Link>
      </div>
    </Container>
  );
}
