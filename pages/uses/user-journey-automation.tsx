import React, { useState, useEffect } from "react";
import styled from "@emotion/styled";

import Hero from "src/shared/legacy/Hero";
import Examples from "src/shared/legacy/Examples";
import FeatureCallouts from "src/shared/legacy/FeatureCallouts";
import Button from "src/shared/legacy/Button";
import Nav from "src/shared/legacy/nav";
import Footer from "src/shared/legacy/Footer";
import DemoBlock from "src/shared/legacy/DemoBlock";
import GraphicCallout from "src/shared/legacy/GraphicCallout";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Automate User Journeys in Minutes",
        description:
          "Build out user-behavior driven flows for your product that are triggered by events sent from your app or third party integrations.",
      },
    },
  };
}

const examples = [
  {
    title: "Handle failed payments", // Alt: Handle involuntary churn
    steps: [
      {
        icon: "/icons/brands/stripe.jpg",
        description: "Stripe Webhook Trigger",
        action: (
          <>
            When <code>charge.failed</code> is received
          </>
        ),
      },
      {
        icon: "/icons/brands/mongodb.svg",
        description: "Run custom code",
        action: "Downgrade the user's plan in the database",
      },
      {
        icon: "/icons/brands/intercom.png",
        description: "Run custom code",
        action: "Notify Customer Success team in Intercom",
      },
    ],
  },
  {
    title: "Intelligent activation drip campaign",
    steps: [
      {
        icon: "/icons/webhook.svg",
        description: "Custom Event",
        action: "When a user signs up",
      },
      {
        icon: "/icons/delay.png",
        description: "Delay",
        action: "Wait 24 hours",
      },
      {
        icon: "/icons/conditional.webp",
        description: "Conditional logic",
        action: "If user does not activate",
      },
      {
        icon: "/icons/brands/sendgrid.png",
        description: "Run custom code",
        action: "Send onboarding email",
      },
    ],
  },
  {
    title: "Running scripts from internal tools",
    steps: [
      {
        icon: "/icons/brands/retool.jpg",
        description: "Retool Resource Request",
        action: "When a form is submitted",
      },
      {
        icon: "/icons/brands/javascript.png",
        description: "Run custom code",
        action: "Run a backfill of user data",
      },
    ],
  },
];

const calloutFeatures = [
  {
    topic: "Any Source",
    title: "Connect Anything",
    description:
      "Send data via Inngest's webhooks, from your code with our API, or use one of Inngest's built-in integrations, ",
    image: "/assets/screenshots/sources.png",
    // TODO - Link to sources page (integrations, webhooks, api keys/SDKs)
  },
  {
    topic: "Developer UX",
    title: "Intuitive Developer Tooling",
    description:
      "A CLI that gets out your way and makes the hard stuff easy. Create, test, and deploy functions in minutes.",
    image: "/assets/homepage/cli-3-commands.png",
    // TODO - Link to CLI or "for developers"/developer-ux page
  },
  {
    topic: "Out-of-the-box Power",
    title: "Conditional Logic, Delays, & Automate Retries",
    description:
      "Use minimal declarative configuration to create complex flows that can delay for days, conditionally run based on data, and automatically retry failed functions.",
    image: "/assets/use-cases/conditional-logic.png",
    // TODO - Link to features page section
  },
  {
    topic: "Step Functions",
    title: "Chain Functions Together",
    description:
      "Break your code into logical steps and run them in parallel, in sequence, or conditionally based on the output of previous steps.",
    image: "/assets/use-cases/step-function.png",
    // TODO - Link to features page section on step functions
  },
];

const page = {
  hero: {
    headline: (
      <>
        Automate{" "}
        <span className="md:whitespace-nowrap	gradient-text-ltr gradient-from-iris-60 gradient-to-cyan">
          User Journeys
        </span>
        <br />
        in Minutes.
      </>
    ),
    subheadline:
      "Build out user-behavior driven flows for your product that are triggered by events sent from your app or third party integrations.",
    primaryCTA: {
      href: `${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=user-journey`,
      text: "Get started for free",
    },
    secondaryCTA: {
      href: "/contact?ref=user-journey",
      text: "Request a demo",
    },
  },
};

export default function Template() {
  return (
    <>
      <Nav sticky={true} />

      <Hero className="hero-gradient" {...page.hero} />

      <Examples
        heading={
          <>
            What some of our customers have{" "}
            <span className="underline italic text-green-700 decoration-sky-500">
              shipped
            </span>
          </>
        }
        examples={examples}
      />

      <div className="container mx-auto max-w-5xl my-24">
        <div className="text-center px-6 max-w-2xl mx-auto">
          <h2 className="text-4xl mb-6">
            <span className="gradient-text gradient-text-ltr gradient-from-pink gradient-to-orange">
              Why?
            </span>
          </h2>
          <p className="text-md">
            User journey automation is high impact, but also high effort.
            Delivering personalized, unique experiences for your customers
            during onboarding, sales, or every billing period makes the
            difference between a good and great product.
          </p>
          <p className="text-md">
            Inngest takes out the complex part of tracking state, coordinating
            between different events, conditionally triggering certain flows and
            allows you to just focus on building an improved, tailored
            experience for your customers.
          </p>
        </div>
      </div>

      <GraphicCallout
        heading="Trigger your code directly from Retool"
        description="See how you can easily run existing code and scripts right from Retool with the power and flexibility of Inngest"
        image="/assets/use-cases/guide-retool-inngest.png"
        cta={{
          href: "/docs/guides/trigger-your-code-from-retool?ref=user-journey-graphic-callout",
          text: "Read the guide",
        }}
        style={{
          backgroundImage:
            "linear-gradient(135deg, rgba(171, 220, 255, 0.5) 0%, rgba(3, 150, 255, 0.5) 100%)",
        }}
      />

      <FeatureCallouts
        heading={
          <>
            Deliver{" "}
            <span className="gradient-text gradient-text-ltr gradient-from-cyan gradient-to-pink">
              personalized,&nbsp;unique
            </span>
            <br />
            experiences for your users
          </>
        }
        features={calloutFeatures}
        cta={{
          href: `${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=user-journey-features`,
          text: "Get started building now →",
        }}
      />

      <DemoBlock
        headline="Inngest provides the tools for any automation"
        description="Skip the boilerplate and get right to the heart of the matter: writing code that helps your business achieve its goals."
      />

      <Footer />
    </>
  );
}
