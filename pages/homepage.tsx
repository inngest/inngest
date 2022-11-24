import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Head } from "next/document";
import HomePatternsCheck from "src/shared/Icons/HomePatternsCheck";
import ArrowRight from "src/shared/Icons/ArrowRight";

// import Footer from "../shared/Footer";

import Button from "src/shared/Button";
import CodeWindow from "src/shared/CodeWindow";
import Discord from "src/shared/Icons/Discord";
import SendEventsImg from "src/shared/Home/HomeImg/SendEventsImg";

import Header from "src/shared/Home/Header";
import Hero from "src/shared/Home/Hero";
import EventDriven from "src/shared/Home/EventDriven";
import DevUI from "src/shared/Home/DevUI";
import OutTheBox from "src/shared/Home/OutTheBox";
import Footer from "src/shared/Home/Footer";

import Patterns from "src/shared/Home/Patterns";
import Container from "src/shared/Home/Container";

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

      <Container className="mt-40">
        <h2 className="text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
          The complete platform, fully managed for you
        </h2>
        <p className="text-slate-300 font-light max-w-md lg:max-w-xl text-sm lg:text-base leading-5 lg:leading-7">
          Our serverless platform provides all the observability, tools, and
          features so you can focus on just building your product.
        </p>
      </Container>

      <Container className="mt-20 flex items-start gap-16">
        <div className="w-1/2">
          <h4 className="text-white text-2xl mb-2">
            Full observability at your fingertips
          </h4>
          <p className="text-sm text-slate-400 max-w-lg ">
            Our platform surfaces failures so you can fix them faster than ever.
            You shouldn’t spend half your day parsing logs.
          </p>
          <ul className="flex flex-col gap-2 mt-8">
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> Full logs - Functions & Events
            </li>
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> Metrics
            </li>
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> Debugging tools
            </li>
          </ul>
        </div>
        <div className="w-1/2">
          <h4 className="text-white text-2xl mb-2">
            We build the hard stuff for you{" "}
          </h4>
          <p className="text-sm text-slate-400 max-w-lg">
            Every feature that you need to run your code reliably, included in
            every pricing plan.
          </p>
          <ul className="flex flex-col gap-2 mt-8">
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> Automatic retries of failed functions
            </li>
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> Event replay
            </li>
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> Function & event versioning
            </li>
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> TypeScript type generation from events
            </li>
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> Throttling
            </li>
            <li className="flex gap-1 text-slate-200 items-center">
              <HomePatternsCheck /> Idempotency
            </li>
          </ul>
        </div>
      </Container>

      <Container className="mt-40">
        <h2 className="text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
          Inngest SDK Roadmap
        </h2>
        <p className="text-slate-300 max-w-md lg:max-w-xl text-sm lg:text-base leading-5 lg:leading-7">
          What we've built and what's up next.
        </p>
      </Container>

      <Container className="flex gap-8 rounded-lg mt-12">
        <div className="w-1/3 ">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Future</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Additional platform support (AWS Lambda, Supabase, Deno)
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Additional framework support (Remix, RedwoodJS)
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Testing APIs
            </li>
          </ul>
        </div>
        <div className="w-1/3">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Now</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Step delays, conditional expressions, & event-coordination
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Inngest Cloud deploy
            </li>
          </ul>
        </div>
        <div className="w-1/3">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Launched</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-800/80 rounded text-base overflow-hidden">
              <div className="flex items-center px-6 py-4">
                Step functions{" "}
                <span className="px-1.5 py-1 font-medium leading-none text-white bg-indigo-500 rounded text-xs ml-2">
                  New
                </span>
              </div>
              <div className="flex flex-wrap px-4 py-2 bg-slate-900">
                <span className="bg-cyan-600 text-slate-200 text-xs font-medium leading-none px-2 py-1 rounded-full">
                  Frameworks
                </span>
              </div>
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Create event-driven and scheduled functions
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Send events
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              TypeScript: Event Type generation and sync (
              <a
                className="text-indigo-400"
                href="/docs/typescript?ref=features-sdk-roadmap"
              >
                docs
              </a>
              )
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Typescript support, including generics
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              <div>
                <a
                  className="text-indigo-400"
                  href="/docs/frameworks/nextjs?ref=features-sdk-roadmap"
                >
                  Next.js
                </a>{" "}
                &amp;{" "}
                <a
                  className="text-indigo-400"
                  href="/docs/frameworks/express?ref=features-sdk-roadmap"
                >
                  Express.js
                </a>{" "}
                support
              </div>
              <div className="flex flex-wrap mt-3">
                <span className="bg-cyan-600 text-slate-200 text-xs font-medium leading-none px-2 py-1 rounded-full">
                  Frameworks
                </span>
              </div>
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              <a
                className="text-indigo-400"
                href="/docs/deploy/cloudflare?ref=features-sdk-roadmap"
              >
                Cloudflare Pages
              </a>{" "}
              support
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-base px-6 py-4">
              Inngest local dev server integration
            </li>
          </ul>
        </div>
      </Container>

      <Footer />
    </div>
  );
}
