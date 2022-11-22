import React, { useEffect, useState } from "react";
import { Head } from "next/document";

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

      <div className="max-w-container-desktop m-auto mb-12 px-10 2xl:-mt-20 relative">
        <h2 className="text-slate-50 font-medium text-5xl mb-7  tracking-tighter ">
          Event driven, made simple
        </h2>
        <p className="text-slate-300 font-light max-w-xl leading-7">
          Our dev server runs on your machine providing you instant feedback and
          debugging tools so you can build serverless functions with events like
          never before possible.
        </p>
      </div>

      <div className="bg-gradient-to-r from-slate-1000/0  to-slate-900">
        <div className="max-w-container-desktop m-auto flex px-10 relative">
          <div className="py-16">
            <h3 className="text-lg xl:text-2xl text-slate-50 mb-3">
              Write code, send events
            </h3>
            <p className="text-slate-400 font-light text-sm max-w-sm leading-7">
              Use the Inngest SDK to define functions that are triggered by
              events from your app or anywhere on the internet.
            </p>
            <code className="text-xs mr-5">
              <span className="text-slate-500 mr-2">$</span>
              npm install inngest
            </code>
          </div>
          <SendEventsImg />
        </div>
      </div>

      <section>
        {/* Content layout */}
        <div className="mx-auto my-28 px-10 lg:px-16 max-w-5xl">
          <header className="my-12 text-center">
            <h2 className="text-4xl">Tools For Lightspeed Development*</h2>
            <p className="mx-auto my-4 max-w-md">
              Our dev server runs on your machine providing you instant feedback
              and debugging tools so you can build serverless functions with
              events like never before possible.
            </p>
            <p className="mx-auto mt-6 max-w-md text-xs text-slate-500">
              <em>* actual words a customer used</em>
            </p>
          </header>
          <div>
            <CodeWindow
              theme="dark"
              snippet="$ npx inngest-cli dev"
              type="terminal"
              showTitleBar={false}
              className="mx-auto w-44 relative sm:z-20 self-center sm:self-start shadow-md"
            />
            <img
              src="/assets/homepage/placeholders/dev-ui-screenshot-nov-2022.png"
              className="mt-4 rounded-t-md"
            />
          </div>
        </div>
      </section>

      <Footer />
    </div>
  );
}
