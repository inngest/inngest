import React from "react";
import styled from "@emotion/styled";
import Link from "next/link";

import Header from "../shared/Header";
import Footer from "../shared/Footer";
import Container from "../shared/layout/Container";

export const getStaticProps: GetStaticProps = async (ctx) => {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Patterns: Async + Event-Driven",
        description:
          "A collection of software architecture patterns for asynchronous flows",
        image: "/assets/patterns/og-image-patterns.jpg",
      },
    },
  };
};

export default function Patterns() {
  return (
    <div>
      <Header />

      <div>
        <Container className="pt-20">
          <div className="grid grid-cols-2">

            <div className="md:bg-slate-900/20 rounded-lg px-8 pb-4">
              <p className="text-xl text-slate-100 pb-6 font-bold">
                What do you need to build?
              </p>
              <textarea
                placeholder="Create a function that..."
                className="width-100 bg-slate-800/50 backdrop-blur-md border border-slate-700/30 rounded text-slate-200 shadow-lg w-full h-52"
              />

              <p className="text-xs text-slate-500 mt-12 mb-4 text-center">Your history:</p>

              <div className="text-xs text-slate-500">
                <p>NO HISTORY HOMIE</p>
              </div>

              <p className="text-xs text-slate-500 mt-12 mb-4 text-center">Or use an example:</p>

              {EXAMPLE_PROMPTS.map((prompt) => {
                return (
                  <div className="border border-slate-700/30 rounded text-slate-300 shadow-lg text-sm mb-4 hover:bg-slate-50 group/card transition-all hover:border-slate-200 cursor-pointer">
                    <div className="px-6 py-4 lg:px-8 lg:py-6 h-full flex flex-col justify-between group-hover/card:text-slate-700  ">
                      <p>{prompt.prompt}</p>
                    </div>
                    <div className="flex flex-wrap gap-2 bg-slate-950 group-hover/card:bg-slate-100  rounded-b-lg py-3 px-6 group-hover/card:border-slate-200 transition-all">
                      {prompt.tags.map((t) => (
                        <span
                          key={t}
                          className="py-1 px-2 rounded bg-slate-800 text-slate-300 group-hover/card:bg-slate-200 group-hover/card:text-slate-500 transition-all font-medium text-xs"
                        >
                          {t}
                        </span>
                      ))}
                    </div>
                  </div>
                );
              })}

              <h1 className="text-3xl lg:text-5xl text-white mt-12 md:mt-20 font-semibold tracking-tight">
                LLM-driven workflows
              </h1>
              <p className="my-4 text-indigo-200 max-w-xl">
                Use Inngest's LLM prompts to create reliable, durable step functions deployable to any provider.
              </p>

            </div>

            <div></div>
          </div>

          <section className="flex flex-col gap-12">
            {/* Content layout */}
          </section>
        </Container>
      </div>
      <Footer />
    </div>
  );
}

const EXAMPLE_PROMPTS = [
  {
    tags: ["OpenAI", "Parallelism"],
    prompt:
      "Create a function that uses OpenAI to summarize text.  It should take a long string of text, splits the text into chunks, uses openAI to summarize the chunks in parallel, then summarizes all summaries.",
    answer: {
      description: `Here we create a function called "Summarize text" that takes a long string of text, splits the text into chunks, uses openAI to summarize the chunks in parallel, then summarizes all summaries. We use step tooling to run as many actions in parallel as possible and provide retries and durability to each.`,
      references: ["https://www.inngest.com/docs/functions/multi-step"],
      code: `inngest.createFunction(
  { name: "Summarize text" },
  { event: "app/text.summarize" },
  async ({ event, step }) => {
    const chunks = splitTextIntoChunks(event.data.text);

    const summaries = await Promise.all(
      chunks.map((chunk) =>
        step.run("Summarize chunk", () => summarizeChunk(chunk))
      )
    );

    await step.run("Summarize summaries", () => summarizeSummaries(summaries));
  }
);`,
    },
  },

  {
    tags: ["Complex", "Event coordination", "Example: Doordash app"],
    prompt:
      "Create a function triggered by an order being created. It should charge the customer for the product in the order, failing if the charge did not succeed. We then wait for the order to be picked up. If it wasn't picked up within an hour, refund and notify the user. If the order was picked up, send a push notification to the user that it's been collected. We wait again for the order to be delivered this time. If it hasn't been delivered within an hour, refund and notify the user the same as before. If it does get delivered, send a push notification that the order has been delivered, wait 30 minutes, then another push notification asking them to rate their meal.",
    answer: {
      description: `Here we create a function called "Order processing" triggered by an app/order.created event. It charges the customer for the product in the order, failing if the charge did not succeed. We then wait for the order to be picked up. If it wasn't picked up within an hour, refund and notify the user. If the order was picked up, send a push notification to the user that it's been collected. We wait again for the order to be delivered this time. If it hasn't been delivered within an hour, refund and notify the user the same as before. If it does get delivered, send a push notification that the order has been delivered, wait 30 minutes, then another push notification asking them to rate their meal. We use step tooling to run as many actions in parallel as possible and provide retries and durability to each.`,
      references: ["https://www.inngest.com/docs/functions/multi-step"],
      code: `inngest.createFunction(
  { name: "Order processing" },
  { event: "app/order.created" },
  async ({ event, step }) => {
    await step.run("Charge customer", () =>
      chargeCustomer(event.data.customerId, event.data.productId)
    );

    const orderPickedUp = await step.waitForEvent(
      "app/order.pickedup",
      {
        timeout: "1h",
        match: "data.orderId",
      }
    );

    if (!orderPickedUp) {
      await step.run("Refund customer", () =>
        refundCustomer(event.data.customerId, event.data.productId)
      );

      await step.run("Notify user", () =>
        notifyUser(event.data.customerId, "Your order was not picked up")
      );

      return;
    }

    await step.run("Notify user", () =>
      notifyUser(event.data.customerId, "Your order has been picked up")
    );

    const orderDelivered = await step.waitForEvent(
      "app/order.delivered",
      {
        timeout: "1h",
        match: "data.orderId",
      }
    );

    if (!orderDelivered) {
      await step.run("Refund customer", () =>
        refundCustomer(event.data.customerId, event.data.productId)
      );

      await step.run("Notify user", () =>
        notifyUser(event.data.customerId, "Your order was not delivered")
      );
      return;
    }

    await step.run("Notify user", () =>
      notifyUser(event.data.customerId, "Your order has been delivered")
    );

    await step.sleep("30m");

    await step.run("Notify user", () =>
      notifyUser(event.data.customerId, "Please rate your meal")
    );
  }
);`,
    },
  },
];
