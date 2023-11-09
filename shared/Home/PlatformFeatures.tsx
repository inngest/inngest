import Link from "next/link";
import Container from "../layout/Container";
import clsx from "clsx";

import Heading from "./Heading";

const highlights = [
  {
    title: "Ship reliable code",
    description:
      "All functions are retried automatically. Manage concurrency, rate limiting and backoffs in code within your function.",
    img: "/assets/homepage/platform/ship-code.png",
  },
  {
    title: "Powerful scheduling",
    description:
      "Enqueue future work, sleep for months, and dynamically cancel jobs without managing job state or hacking APIs together.",
    img: "/assets/homepage/platform/powerful-scheduling.png",
  },
  {
    title: "Replay functions at any time",
    description:
      "Forget the dead letter queue. Replay functions that have failed, or replay functions in your local environment to debug issues easier than ever before.",
    img: "/assets/homepage/platform/replay-functions.png",
  },
];

export default function PlatformFeatures() {
  return (
    <Container className="my-44 tracking-tight">
      <Heading
        title="Giving developers piece of mind"
        lede="Inngest gives you everything you need with sensible defaults."
        className="mx-auto max-w-3xl text-center"
      />

      <div className="my-24 mx-auto max-w-6xl flex flex-col gap-8">
        {highlights.map(({ title, description, img }, idx) => (
          <div
            key={idx}
            className={clsx(
              `flex flex-col items-stretch bg-slate-950 border rounded-xl p-2.5 border-slate-900`,
              idx % 2 === 0 ? `lg:flex-row-reverse` : `lg:flex-row`
            )}
          >
            <div className=" px-6 lg:px-10 py-6 lg:py-12 flex flex-col justify-center">
              <h3 className="text-xl text-indigo-50 font-semibold">{title}</h3>
              <p className="my-1.5 text-sm lg:text-base text-indigo-200">
                {description}
              </p>
            </div>
            <div className="bg-slate-1000 w-full rounded flex items-center justify-center">
              <img
                src={img}
                alt={`Graphic for ${title}`}
                className="w-full max-w-[600px] m-auto"
              />
            </div>
          </div>
        ))}
      </div>
    </Container>
  );
}
