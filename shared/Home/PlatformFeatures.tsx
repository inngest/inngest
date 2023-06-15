import Link from "next/link";
import Container from "../layout/Container";

import Heading from "./Heading";

const highlights = [
  {
    title: "Ship reliable code",
    description:
      "All functions are retried automatically. Manage concurrency, rate limiting and backoffs in code within your function.",
    img: "/assets/homepage/platform/reliable-code.svg",
  },
  {
    title: "Powerful scheduling",
    description:
      "Enqueue future work, sleep for months, and dynamically cancel jobs without managing job state or hacking APIs together.",
    img: "/assets/homepage/platform/powerful-scheduling.svg",
  },
  {
    title: "Replay functions at any time",
    description:
      "Forget the dead letter queue. Replay functions that have failed, or replay functions in your local environment to debug issues easier than ever before.",
    img: "/assets/homepage/platform/replay-functions.svg",
  },
];

export default function PlatformFeatures() {
  return (
    <Container className="my-44 tracking-tight">
      <Heading
        title="We’ve built the hard stuff for you"
        lede="Inngest gives you everything you need with sensible defaults."
        className="mx-auto max-w-3xl text-center"
      />

      <div className="my-24 mx-auto max-w-6xl flex flex-col gap-12">
        {highlights.map(({ title, description, img }, idx) => (
          <div
            key={idx}
            className="grid md:grid-cols-3 gap-16 justify-between items-center rounded-lg"
          >
            <div className={`${idx % 2 === 0 ? "" : "md:col-start-3 order-2"}`}>
              <h3 className="text-xl text-indigo-50 font-semibold">{title}</h3>
              <p className="my-1.5 text-indigo-200">{description}</p>
            </div>
            <img
              src={img}
              className={`w-full max-h-72 px-12 pointer-events-none md:col-span-2 ${
                idx % 2 === 0 ? "" : "md:col-start-0 md:order-1"
              }`}
              alt={`Graphic for ${title}`}
            />
          </div>
        ))}
      </div>

      <div className="flex items-center justify-center">
        <Link
          href="/product/how-inngest-works?ref=homepage-run-anywhere"
          className="rounded-md font-medium px-9 py-3.5 bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
        >
          Start Building For Free
        </Link>
      </div>
    </Container>
  );
}
