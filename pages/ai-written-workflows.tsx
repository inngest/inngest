import React, { useEffect, useState, useRef } from "react";
import styled from "@emotion/styled";
import Link from "next/link";
import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";

import { MDXRemote } from "next-mdx-remote";

import Header from "../shared/Header";
import Footer from "../shared/Footer";
import Container from "../shared/layout/Container";
import { Button } from "src/shared/Button";
import Arrow from "src/shared/Icons/Arrow";
import ArrowRight from "src/shared/Icons/ArrowRight";
import Twitter from "src/shared/Icons/Twitter";

export const getStaticProps = async () => {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Write Inngest functions using GPT",
        description:
          "Use GPT to write Inngest workflows and functions via our SDK",
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

export default function InngestGPT() {
  const [selected, setSelected] = useState<Selected | null>(EXAMPLE_PROMPTS[0]);
  const [history, setHistory] = useState<Selected[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const resultRef = useRef(null);

  useEffect(() => {
    let data = "[]";
    try {
      data = window?.localStorage?.getItem("ai-sdk-history") || "[]";
    } catch (e) {
      return;
    }
    const items = JSON.parse(data) as Selected[];
    setHistory(items);
    if (items.length > 0) {
      setSelected(items[0]);
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
      const newHistory = [{ ...data, prompt: message }].concat(history);
      setHistory(newHistory);
      setSelected({ ...data, prompt: message });
      window?.localStorage?.setItem(
        "ai-sdk-history",
        JSON.stringify(newHistory)
      );
      resultRef.current.scrollIntoView({ block: "start", behavior: "smooth" });
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
          backgroundImage: "url(/assets/pricing/table-bg.png)",
          backgroundPosition: "center -30px",
          backgroundRepeat: "no-repeat",
          backgroundSize: "1800px 1200px",
        }}
      >
        <Container className="pb-16">
          <h1 className="text-3xl lg:text-5xl text-white mt-12 md:mt-20 xl:mt-32 font-semibold tracking-tight text-center">
            GPT-driven workflows
          </h1>
          <p className="mt-4 text-indigo-200 max-w-xl text-center m-auto xl:mb-32">
            Use Inngest's GPT prompts to create reliable, durable step functions
            deployable to any provider.
          </p>

          <div className="max-w-3xl m-auto my-20">
            <div className=" bg-white rounded-lg shadow-lg overflow-hidden">
              <div className="py-6 px-6 w-full">
                <textarea
                  disabled={loading}
                  placeholder="Create a function that..."
                  className="  backdrop-blur-md border border-slate-700/30 rounded-md text-slate-700 font-medium w-full h-52 focus:outline-none"
                  onChange={(e) => setMessage(e.target.value)}
                />
              </div>
              <div className="flex justify-between items-center px-4 py-2 bg-slate-100">
                <span className="text-sm font-medium text-slate-700">
                  Powered by OpenAI
                </span>
                <a
                  onClick={onSubmit}
                  className={`group inline-flex items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2 text-white ${
                    loading
                      ? "bg-slate-500"
                      : "bg-indigo-500 hover:bg-indigo-400 text-white"
                  } transition-all`}
                >
                  {loading ? (
                    <div className="flex items-center">
                      <span>Generating</span>
                      <svg
                        aria-hidden="true"
                        className="w-4 h-4 ml-3 text-slate-400 animate-spin dark:text-gray-600 fill-white"
                        viewBox="0 0 100 101"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M100 50.5908C100 78.2051 77.6142 100.591 50 100.591C22.3858 100.591 0 78.2051 0 50.5908C0 22.9766 22.3858 0.59082 50 0.59082C77.6142 0.59082 100 22.9766 100 50.5908ZM9.08144 50.5908C9.08144 73.1895 27.4013 91.5094 50 91.5094C72.5987 91.5094 90.9186 73.1895 90.9186 50.5908C90.9186 27.9921 72.5987 9.67226 50 9.67226C27.4013 9.67226 9.08144 27.9921 9.08144 50.5908Z"
                          fill="currentColor"
                        />
                        <path
                          d="M93.9676 39.0409C96.393 38.4038 97.8624 35.9116 97.0079 33.5539C95.2932 28.8227 92.871 24.3692 89.8167 20.348C85.8452 15.1192 80.8826 10.7238 75.2124 7.41289C69.5422 4.10194 63.2754 1.94025 56.7698 1.05124C51.7666 0.367541 46.6976 0.446843 41.7345 1.27873C39.2613 1.69328 37.813 4.19778 38.4501 6.62326C39.0873 9.04874 41.5694 10.4717 44.0505 10.1071C47.8511 9.54855 51.7191 9.52689 55.5402 10.0491C60.8642 10.7766 65.9928 12.5457 70.6331 15.2552C75.2735 17.9648 79.3347 21.5619 82.5849 25.841C84.9175 28.9121 86.7997 32.2913 88.1811 35.8758C89.083 38.2158 91.5421 39.6781 93.9676 39.0409Z"
                          fill="currentFill"
                        />
                      </svg>
                    </div>
                  ) : (
                    "Create your function"
                  )}
                </a>
              </div>
            </div>
            {error !== "" && (
              <p className="text-center text-sm px-4 py-1.5 m-auto text-red-700 font-medium mt-4 bg-red-50/90 rounded-full inline-flex justify-self-center ">
                {error}
              </p>
            )}
          </div>

          <h4 className="text-center mb-8 text-slate-100 text-base">
            Or use an example:
          </h4>
          <div className="grid lg:grid-cols-3 gap-6  mb-10">
            {EXAMPLE_PROMPTS.map((prompt, i) => {
              return (
                <PromptUI
                  key={i}
                  prompt={prompt}
                  selected={selected}
                  onClick={() => setSelected(prompt)}
                />
              );
            })}
          </div>

          <p className="text-sm text-center md:text-left text-slate-200 mb-10 lg:mb-20 gap-2 rounded-lg border border-slate-100/20 py-2 flex flex-col items-center md:flex-row pr-4 pl-2 max-w-[890px] m-auto ">
            <span className="bg-slate-100 mt-2 mb-1 md:mt-0 md:mb-0 md:mr-2 rounded px-2.5 py-1 text-sm font-medium tracking-tight  text-slate-800 shadow flex-shrink-0 ">
              Disclaimer
            </span>
            <span>
              The outputs of this system may be unpredictable and are not
              guaranteed to be perfect code or morally conscious.
            </span>
          </p>

          <div
            className="lg:grid grid-cols-5 gap-12 scroll-mt-32"
            ref={resultRef}
          >
            <div className=" rounded-lg pb-4 col-span-2 overflow-hidden">
              <p className="text-lg text-white px-4 font-medium  mb-4">
                Your history
              </p>

              <div className="text-xs text-slate-700 px-4 ">
                {history.length === 0 ? (
                  <p className="text-slate-300 text-sm leading-relaxed">
                    You haven't submitted anything yet. Either use the form
                    above, or check out one of our examples.
                  </p>
                ) : (
                  <ul className="flex flex-col gap-4">
                    {history.map((prompt: Selected, i) => {
                      return (
                        <PromptUI
                          key={i}
                          prompt={prompt}
                          selected={selected}
                          onClick={() => setSelected(prompt)}
                          variant="history"
                        />
                      );
                    })}
                  </ul>
                )}
              </div>
            </div>

            {selected ? <Output selected={selected} /> : null}
          </div>
          <div className="w-full flex mt-24 justify-center gap-4">
            <Button
              href="/blog/ai-personalization-and-the-future-of-developer-docs"
              variant="secondary"
            >
              Read how this page works
              <ArrowRight className="group-hover:translate-x-1 transition-transform duration-150  -mr-1.5" />
            </Button>
            <Button href="https://twitter.com/intent/tweet?text=hello&url=https://inngest.com/ai-written-workflows">
              <span className="mr-1.5">Tweet this page</span>
              <Twitter className="mt-0.5" />
            </Button>
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
  variant = "example",
}: {
  prompt: Selected;
  selected?: Selected;
  variant?: "example" | "history";
  onClick: () => void;
}) => {
  const isSelected = selected?.prompt === prompt.prompt;

  return (
    <div
      className={`text-center rounded-lg text-slate-300 shadow-lg text-sm hover:scale-105 origin-center origin group/card transition-all cursor-pointer bg-slate-900 px-6 py-4 lg:px-8 lg:py-6  ${
        isSelected && "bg-slate-50 scale-105"
      }`}
      onClick={() => onClick()}
    >
      {variant === "example" && (
        <p
          className={`font-semibold text-lg text-white mb-4 ${
            isSelected && "text-indigo-600"
          }`}
        >
          {prompt.title}
        </p>
      )}
      {variant === "history" && (
        <p
          className={`text-left ${isSelected && "text-slate-800 font-medium"}`}
        >
          {prompt.prompt}
        </p>
      )}
      {prompt.tags && (
        <div className={`flex flex-wrap gap-2 transition-all justify-center`}>
          {prompt?.tags?.map((t) => (
            <span
              key={t}
              className={`py-1 px-2 rounded bg-slate-800 text-slate-300 transition-all font-medium text-xs ${
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
    <div className="col-span-3 col-start-3">
      <div className="bg-white rounded-lg shadow-lg overflow-hidden mt-8 lg:mt-auto">
        <p className="text-sm text-slate-600 font-medium bg-slate-100 py-3 px-6">
          Prompt
        </p>
        <p className=" text-slate-600 leading-relaxed text-base py-4 px-6 lg:py-6 lg:px-12">
          {selected.prompt}
        </p>
      </div>

      <div className="bg-slate-900/80 rounded-lg overflow-hidden mt-4 lg:mt-8">
        <h3 className="text-lg text-white py-6 px-8 bg-slate-800/60">
          Generated Inngest function
        </h3>
        <div className="p-4 lg:p-6">
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

          <p
            className="mb-4 mt-8 text-slate-200"
            dangerouslySetInnerHTML={{
              __html: markdownToHTML(selected.reply.description),
            }}
          ></p>
          <div className="border-t pt-8 border-slate-800 flex flex-col gap-4">
            <p className="text-white">Want to learn more?</p>
            <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
              <a
                href="/docs/quick-start"
                className="bg-slate-800 rounded-lg px-6 py-5 hover:bg-slate-700/80 group block"
              >
                <h4 className="text-white">Quick start guide</h4>
                <span className="text-sm text-indigo-400 flex items-center mt-2">
                  Read the docs{" "}
                  <ArrowRight className="group-hover:translate-x-1 transition-transform duration-150  -mr-1.5" />
                </span>
              </a>
              <a
                href="/docs/functions"
                className="bg-slate-800 rounded-lg px-6 py-5 hover:bg-slate-700/80 group block"
              >
                <h4 className="text-white">Writing functions</h4>
                <span className="text-sm text-indigo-400 flex items-center mt-2">
                  Read the docs{" "}
                  <ArrowRight className="group-hover:translate-x-1 transition-transform duration-150  -mr-1.5" />
                </span>
              </a>
              <a
                href="/docs/events"
                className="bg-slate-800 rounded-lg px-6 py-5 hover:bg-slate-700/80 group block"
              >
                <h4 className="text-white">Sending Events</h4>
                <span className="text-sm text-indigo-400 flex items-center mt-2">
                  Read the docs{" "}
                  <ArrowRight className="group-hover:translate-x-1 transition-transform duration-150  -mr-1.5" />
                </span>
              </a>
            </div>
          </div>
          <h3 className="text-base text-white pt-8 pb-2">References:</h3>
          <ul className="list-disc text-slate-200 ml-4 pb-8">
            {selected.reply.references.map((r) => (
              <li key={r} className="">
                <a className="text-indigo-400" href={r}>
                  {r}
                </a>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
};

const markdownToHTML = (markdown: string): string => {
  return markdown
    .replaceAll(/_(.+)_/g, `<em>$1</em>`)
    .replaceAll(
      /`(.+)`/g,
      `<code class="bg-slate-800 text-white text-sm font-semibold py-0.5 px-1.5">$1</code>`
    );
};

const EXAMPLE_PROMPTS = [
  {
    tags: ["OpenAI", "Parallelism"],
    title: "LLM Summarization",
    prompt:
      "Create a function that uses OpenAI to summarize text.  It should take a long string of text, splits the text into chunks, uses openAI to summarize the chunks in parallel, then summarizes all summaries.",
    reply: {
      description: `Here we create a function called "_Summarize text_" that takes a long string of text, splits the text into chunks, uses openAI to summarize the chunks in parallel, then summarizes all summaries. We use step tooling to run as many actions in parallel as possible and provide retries and durability to each.`,
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
      description: `Here we create a function called "_Order processing_" triggered by an \`app/order.created\` event. It charges the customer for the product in the order, failing if the charge did not succeed. We then wait for the order to be picked up. If it wasn't picked up within an hour, refund and notify the user. If the order was picked up, send a push notification to the user that it's been collected. We wait again for the order to be delivered this time. If it hasn't been delivered within an hour, refund and notify the user the same as before. If it does get delivered, send a push notification that the order has been delivered, wait 30 minutes, then another push notification asking them to rate their meal. We use step tooling to run as many actions in parallel as possible and provide retries and durability to each.`,
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
