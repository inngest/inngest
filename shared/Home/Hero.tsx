import Link from "next/link";
import { ChevronRightIcon } from "@heroicons/react/20/solid";

import Container from "../layout/Container";

/**
 * NOTE - When you update hero copy also update index.tsx's getStaticProps title/description for social & SEO
 */
export default function Hero() {
  return (
    <Container
      className="mt-24 md:mt-36 tracking-tight lg:bg-[url(/assets/homepage/use-case-line-graphic.svg)]"
      style={{
        backgroundSize: "1980px 198px",
        backgroundPosition: "50% 258px",
        backgroundRepeat: "no-repeat",
      }}
    >
      <div className="mb-12 md:mb-28 text-center">
        <h1 className="
          text-4xl md:text-[3.125rem] md:leading-[3.75rem]
          font-bold bg-clip-text
          text-transparent bg-gradient-to-r from-[#E2BEFF] via-white to-[#AFC1FF] drop-shadow
        ">
          Effortless serverless queues,
          <br />
          background jobs, and workflows
        </h1>
        <p className="mt-6 mx-auto max-w-2xl text-lg text-indigo-200 drop-shadow">
          Easily develop serverless workflows in your current codebase, without
          any new infrastructure. Using Inngest, your entire team can ship
          reliable products.
        </p>
      </div>

      <div className="flex flex-col gap-8 pt-12 lg:py-28 items-center justify-center">
        <div>
          <Link
            href="/docs?ref=homepage-hero"
            className="rounded-md font-medium px-11 py-3.5 bg-indigo-500 hover:bg-indigo-400 transition-all text-white whitespace-nowrap flex flex-row items-center"
          >
            Quick Start Guide{" "}
            <ChevronRightIcon className="h-5 group-hover:translate-x-1 relative top-px transition-transform duration-150" />
          </Link>
        </div>
        <Link
          href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=homepage-hero`}
          className="group flex items-center gap-1 rounded-md px-11 py-3.5 bg-transparent transition-all text-indigo-100 border border-transparent hover:border-slate-800 whitespace-nowrap"
        >
          Start Building For Free
        </Link>
      </div>
    </Container>
  );
}
