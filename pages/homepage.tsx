import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Head } from "next/document";
import HomePatternsCheck from "src/shared/Icons/HomePatternsCheck";
import ArrowRight from "src/shared/Icons/ArrowRight";

import Footer from "../shared/Footer";

import Button from "src/shared/Button";
import CodeWindow from "src/shared/CodeWindow";
import Discord from "src/shared/Icons/Discord";
import SendEventsImg from "src/shared/Home/HomeImg/SendEventsImg";

import Header from "src/shared/Home/Header";
import Hero from "src/shared/Home/Hero";

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

      <div className="max-w-container-desktop m-auto mt-20 mb-12 px-10 relative z-10">
        <h2 className="text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
          Event driven, made simple
        </h2>
        <p className="text-slate-300 font-light max-w-md lg:max-w-xl text-sm leading-5 lg:leading-7">
          Add Inngest to your stack in a few lines for code, then deploy to your
          existing provider. You don't have to change anything to get started.
        </p>
      </div>

      <div className="bg-gradient-to-r from-slate-1000/0  to-slate-900 relative z-10">
        <div className="max-w-container-desktop m-auto flex px-10 relative">
          <div className="py-16">
            <h3 className="text-lg xl:text-2xl text-slate-50 mb-3">
              Write code, send events
            </h3>
            <p className="text-slate-400 font-light text-sm max-w-sm leading-7">
              Use the Inngest SDK to define functions that are triggered by
              events from your app or anywhere on the internet.
            </p>
            <code className="text-xs mr-5 text-slate-50 mt-8 inline-block bg-slate-800/50 px-4 py-2 rounded-lg">
              <span className="text-slate-500 mr-2">$</span>
              npm install inngest
            </code>
          </div>
          {/* <SendEventsImg /> */}
        </div>
      </div>

      <div className="max-w-container-desktop m-auto flex px-10 relative gap-4 mt-24">
        <div className=" w-1/2 flex flex-col gap-10 justify-between text-center bg-slate-800/60 rounded-xl py-11 px-16">
          <div>
            <h4 className="text-white text-2xl font-medium tracking-tight mb-2">
              Use with your favorite frameworks
            </h4>
            <p className="text-slate-400">
              Write your code directly within your existing codebase.
            </p>
          </div>
          <div>logos</div>
        </div>
        <div className="w-[86px] h-[84px] relative z-10 mt-16 -mx-[54px]  leading-none flex items-center justify-center rounded-full bg-indigo-500 text-white text-2xl font-medium border-slate-1000 border-8">
          &
        </div>
        <div className=" w-1/2 flex flex-col justify-between text-center bg-slate-800/60 rounded-xl py-11 px-16">
          <div>
            <h4 className="text-white text-2xl font-medium tracking-tight mb-2">
              Deploy functions anywhere
            </h4>
            <p className="text-slate-400">
              Inngest calls your code, securely, as events are received.
              <br />
              Keep shipping your code as you do today.
            </p>
          </div>
          <div>logos</div>
        </div>
      </div>

      <div>
        <div className="max-w-container-desktop m-auto mt-20 mb-12 px-10 relative z-10">
          <h2 className="lg:flex gap-2 items-end text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
            Tools for "lightspeed development*"{" "}
            <span className="inline-block text-sm text-slate-500 tracking-normal ">
              *actual words a customer used
            </span>
          </h2>
          <p className="text-slate-300 font-light max-w-md lg:max-w-xl text-sm leading-5 lg:leading-7">
            Our dev server runs on your machine providing you instant feedback
            and debugging tools so you can build serverless functions with
            events like never before possible.
          </p>
        </div>
      </div>

      <div className="max-w-container-desktop m-auto px-10 relative z-10">
        <div className="absolute inset-0 rounded-lg bg-indigo-500 opacity-20 rotate-1 -z-0 scale-[102%] mx-5"></div>
        <div
          style={{
            backgroundImage: "url(/assets/footer-grid.svg)",
            backgroundSize: "cover",
            backgroundPosition: "left center",
          }}
          className="mt-20 mb-12 p-8 md:p-12 lg:px-16 lg:py-16 bg-indigo-600 rounded-lg shadow-3xl relative z-10"
        >
          <h3 className="text-slate-50 font-medium text-2xl lg:text-3xl xl:text-4xl mb-4 tracking-tighter ">
            Learn the patterns so you can build anything
          </h3>
          <p className="text-slate-200 font-regular max-w-md lg:max-w-xl text-sm leading-5 lg:leading-6">
            We’ve documented the key patterns that devs encounter when building
            background jobs or scheduled jobs - from the basic to the advanced.
            Read the patterns and learn how to create them with Inngest in just
            a few minutes:
          </p>
          <ul className="flex flex-col gap-1.5 md:gap-0 md:flex-row md:flex-wrap max-w-[600px] mt-6 mb-10">
            <li className="text-slate-200 flex text-sm md:w-1/2 md:mb-2">
              <HomePatternsCheck />{" "}
              <span className="ml-2 block">Build reliable webhooks</span>
            </li>
            <li className="text-slate-200 flex text-sm md:w-1/2 md:mb-2">
              <HomePatternsCheck />{" "}
              <span className="ml-2 block">Running functions in parallel</span>
            </li>
            <li className="text-slate-200 flex text-sm md:w-1/2">
              <HomePatternsCheck />{" "}
              <span className="ml-2 block">
                Reliably run critical workflows
              </span>
            </li>
            <li className="text-slate-200 flex text-sm md:w-1/2">
              <HomePatternsCheck />{" "}
              <span className="ml-2 block">
                Building flows for lost customers
              </span>
            </li>
          </ul>
          <a
            href="/patterns"
            className="rounded-full inline-flex text-sm font-medium pl-6 pr-5 py-2 bg-slate-800 hover:bg-indigo-800 transition-all text-white gap-1.5"
          >
            Browse patterns
            <ArrowRight />
          </a>
        </div>
      </div>

      <div className="max-w-container-desktop m-auto mt-20 mb-12 px-10 relative z-10">
        <h2 className="text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
          The complete platform, fully managed for you
        </h2>
        <p className="text-slate-300 font-light max-w-md lg:max-w-xl text-sm leading-5 lg:leading-7">
          Our serverless platform provides all the observability, tools, and
          features so you can focus on just building your product.
        </p>
      </div>

      <Footer />
    </div>
  );
}
