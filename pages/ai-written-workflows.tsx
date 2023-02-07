import React, { useEffect, useState } from "react";
import styled from "@emotion/styled";
import Link from "next/link";
import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";

import Header from "../shared/Header";
import Footer from "../shared/Footer";
import Container from "../shared/layout/Container";

export const getStaticProps = async () => {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Write Inngest functions using GPT",
        description:
          "Use GPT to write Inngest workflows and functions via our SDK",
        image: "/assets/patterns/og-image-patterns.jpg",
      },
    },
  };
};

type Reply = {
  description: string;
  code: string;
  references: string[];
};

type Selected = {
  prompt: string;
  reply: Reply;
  title?: string;
  tags?: string[];
};

export default function Patterns() {
  const [selected, setSelected] = useState<Selected | null>(EXAMPLE_PROMPTS[0]);
  const [history, setHistory] = useState<Selected[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    let data = "[]";
    try {
      data = window?.localStorage?.getItem("ai-sdk-history") || "[]";
    } catch (e) {
      return;
    }
    const items = JSON.parse(data) as Selected[]
    setHistory(items);
    if (items.length > 0) {
      setSelected(items.reverse()[0]);
    }
  }, []);

  const onSubmit = async () => {
    if (loading) {
      return;
    }
    try {
      setLoading(true);
      setError("");
      const result = await fetch("https://inngestabot.deno.dev", {
        method: "POST",
        body: JSON.stringify({
          message,
        }),
        headers: {
          "Content-Type": "application/json",
        },
      });
      const data = await result.json();
      setLoading(false);
      const newHistory = history.concat([{ ...data, prompt: message }]);
      setHistory(newHistory);
      setSelected({ ...data, prompt: message });
      window?.localStorage?.setItem(
        "ai-sdk-history",
        JSON.stringify(newHistory)
      );
    } catch (e) {
      setLoading(false);
      console.warn(e);
      setError("We couldn't generate your function.  Please try again!");
    }
  };

  return (
    <div>
      <Header />

      <div
        style={{
          backgroundImage: "url(/assets/table-bg-20.png)",
          backgroundPosition: "center -30px",
          backgroundRepeat: "no-repeat",
          backgroundSize: "1800px 1200px",
        }}
      >
        <Container className="pt-20">
          <div className="grid grid-cols-2 gap-4">
            <div className="md:bg-slate-900/20 rounded-lg px-8 pb-4">
              <p className="text-xl text-slate-100 pb-6 font-bold">
                What do you need to build?
              </p>
              <textarea
                disabled={loading}
                placeholder="Create a function that..."
                className="width-100 bg-slate-800/50 backdrop-blur-md border border-slate-700/30 rounded-md text-slate-200 shadow-lg w-full h-52"
                onChange={(e) => setMessage(e.target.value)}
              />

              <div className="flex justify-end">
                <a
                  onClick={onSubmit}
                  className={`group flex items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2 text-white ${
                    loading
                      ? "bg-slate-500"
                      : "bg-indigo-500 hover:bg-indigo-400 text-white"
                  } transition-all`}
                >
                  {loading
                    ? "Generating..."
                    : "Create your function via ChatGPT"}
                </a>
              </div>

              {error !== "" && <p className="text-orange-600">{error}</p>}

              <p className="text-xs text-slate-500 mt-12 mb-4 text-center">
                Your history:
              </p>

              <div className="text-xs text-slate-700">
                {history.length === 0 ? (
                  <p className="text-center mt-8">
                    Ya haven't given it a try, yet!
                  </p>
                ) : (
                  history.map((prompt: Selected) => {
                    return (
                      <PromptUI
                        prompt={prompt}
                        selected={selected}
                        onClick={() => setSelected(prompt)}
                      />
                    );
                  })
                )}
              </div>

              <p className="text-xs text-slate-500 mt-12 mb-4 text-center">
                Or use an example:
              </p>

              {EXAMPLE_PROMPTS.map((prompt) => {
                return (
                  <PromptUI
                    prompt={prompt}
                    selected={selected}
                    onClick={() => setSelected(prompt)}
                  />
                );
              })}

              <h1 className="text-3xl lg:text-5xl text-white mt-12 md:mt-20 font-semibold tracking-tight">
                GPT-driven workflows
              </h1>
              <p className="my-4 text-indigo-200 max-w-xl">
                Use Inngest's GPT prompts to create reliable, durable step
                functions deployable to any provider.
              </p>
            </div>

            <div className="md:bg-slate-1000/20 rounded-lg px-8 pb-4">
              {selected ? (
                <Output selected={selected} />
              ) : (
                <div>
                  <p className="text-center my-36 text-xs text-slate-500 mb-4">
                    Enter a prompt or select an example to get started.
                  </p>
                </div>
              )}
            </div>
          </div>
        </Container>
      </div>
      <Footer />
    </div>
  );
}

const PromptUI = ({
  prompt,
  selected,
  onClick,
}: {
  prompt: Selected;
  selected?: Selected;
  onClick: () => void;
}) => {
  const isSelected = selected?.prompt === prompt.prompt;

  return (
    <div
      className={`border border-slate-700/30 rounded text-slate-300 shadow-lg text-sm mb-4 hover:bg-slate-50 group/card transition-all hover:border-slate-200 cursor-pointer bg-slate-900 ${
        isSelected && "bg-slate-50"
      }`}
      onClick={() => onClick()}
    >
      <div
        className={`px-6 py-4 lg:px-8 lg:py-6 h-full flex flex-col justify-between group-hover/card:text-slate-700 ${
          isSelected && "text-slate-700"
        }`}
      >
        {prompt.title && (
          <p className="font-bold pb-4 tex-slate-200">{prompt.title}</p>
        )}
        <p>{prompt.prompt}</p>
      </div>
      {prompt.tags && (
        <div
          className={`flex flex-wrap gap-2 group-hover/card:bg-slate-100  rounded-b-lg py-3 px-6 group-hover/card:border-slate-200 transition-all ${
            isSelected ? "border-slate-200 bg-slate-100" : "bg-slate-950"
          }`}
        >
          {prompt?.tags?.map((t) => (
            <span
              key={t}
              className={`py-1 px-2 rounded bg-slate-800 text-slate-300 group-hover/card:bg-slate-200 group-hover/card:text-slate-500 transition-all font-medium text-xs ${
                isSelected && "text-slate-500 bg-slate-200"
              }`}
            >
              {t}
            </span>
          ))}
        </div>
      )}
    </div>
  );
};

const Output = ({ selected }: { selected: Selected }) => {
  return (
    <div>
      <p className="mb-4 mt-8 text-xs text-slate-500 mb-4">Prompt:</p>
      <p className="mb-4 text-slate-200">{selected.prompt}</p>

      <p className="mb-4 mt-8 text-xs text-slate-500 mb-4">
        Generated Inngest function:
      </p>
      <div className="overflow-x-scroll bg-slate-950/80 backdrop-blur-md border border-slate-800/60 rounded-lg overflow-hidden shadow-lg">
        <h6 className="text-slate-300 w-full bg-slate-950/50 text-center text-xs py-1.5 border-b border-slate-800/50">
          function.ts
        </h6>
        <SyntaxHighlighter
          language="javascript"
          showLineNumbers={false}
          style={syntaxThemeDark}
          codeTagProps={{ className: "code-window" }}
          // className="hello"
          customStyle={{
            backgroundColor: "transparent",
            fontSize: "0.7rem",
            padding: "1.5rem",
          }}
        >
          {selected.reply?.code}
        </SyntaxHighlighter>
      </div>

      <p className="mb-4 mt-8 text-slate-200">{selected.reply.description}</p>

      <p className="text-xs text-slate-500 mt-8 mb-4">References:</p>
      <ul className="list-disc text-slate-200 ml-4">
        {selected.reply.references.map((r) => (
          <li key={r}>
            <a href={r}>{r}</a>
          </li>
        ))}
      </ul>
    </div>
  );
};

const EXAMPLE_PROMPTS = [
  {
    tags: ["OpenAI", "Parallelism"],
    title: "LLM Summarization",
    prompt:
      "Create a function that uses OpenAI to summarize text.  It should take a long string of text, splits the text into chunks, uses openAI to summarize the chunks in parallel, then summarizes all summaries.",
    reply: {
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
    tags: ["Cron", "Fan-out"],
    title: "Weekly reminders",
    prompt:
      "Create a function that runs every Friday at 9AM and queries my database for all users. It should then send an event for each user, where another function listens to that event and sends an email.",
    reply: {
      description: `Here we create a function that runs every Friday at 9AM and queries our database for all users. It then sends an event for each user, where another function listens to that event and sends an email. We use step tooling to run as many actions in parallel as possible and provide retries and durability to each.`,
      references: ["https://www.inngest.com/docs/functions/multi-step"],
      code: `inngest.createFunction(
  { name: "Send weekly email" },
  { cron: "0 9 * * 5" },
  async ({ step }) => {
    const users = await step.run("Get users", () => getUsers());

    await Promise.all(
      users.map((user) =>
        step.run("Send user email event", () =>
          inngest.send("app/user.email.send", {
            data: {
              userId: user.id,
            },
          })
        )
      )
    );
  }
);

inngest.createFunction(
  "Send user email",
  "app/user.email.send",
  async ({ event }) => {
    const user = await getUser(event.data.userId);
    return sendEmail(user.email);
  }
);`,
    },
  },

  {
    tags: ["Complex", "Event coordination", "Example: Doordash app"],
    title: "Delivery app order flow",
    prompt:
      "Create a function triggered by an order being created. It should charge the customer for the product in the order, failing if the charge did not succeed. We then wait for the order to be picked up. If it wasn't picked up within an hour, refund and notify the user. If the order was picked up, send a push notification to the user that it's been collected. We wait again for the order to be delivered this time. If it hasn't been delivered within an hour, refund and notify the user the same as before. If it does get delivered, send a push notification that the order has been delivered, wait 30 minutes, then another push notification asking them to rate their meal.",
    reply: {
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
