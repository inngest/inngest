import { useEffect, useState } from "react";

import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import Quote from "src/shared/Home/Quote";

const CONTACT_KEY =
  "Z-ymc97Dae8u4HHybHknc4DGRb51u6NnTOUaW-qG71ah1ZqsJfRcI6SaHg5APWutNcnMcaN3oZrZky-VQxBIyw";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Chat with solutions engineering",
      },
      designVersion: "2",
    },
  };
}

export default function Contact() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [message, setMessage] = useState("");
  const [teamSize, setTeamSize] = useState("");
  const [disabled, setDisabled] = useState<boolean>(false);
  const [buttonCopy, setButtonCopy] = useState("Send");

  const onSubmit = async (e) => {
    e.preventDefault();
    setDisabled(true);
    setButtonCopy("Sending...");
    let ref = "";
    try {
      const u = new URLSearchParams(window.location.search);
      if (u.get("ref")) {
        ref = u.get("ref");
      }
    } catch (err) {
      // noop
    }
    try {
      await window.Inngest.event(
        {
          name: "contact.form.sent",
          data: { email, name, message, teamSize, ref },
          user: { email, name },
          v: "2023-07-12.1",
        },
        { key: CONTACT_KEY }
      );
      setButtonCopy("Your message has been sent!");
    } catch (e) {
      console.warn("Message not sent");
      setButtonCopy("Message not sent");
      setDisabled(false);
    }
  };

  return (
    <div className="font-sans text-slate-200">
      <Header />
      <Container>
        <main className="m-auto max-w-5xl pt-16 pb-8">
          <header className="pt-12 lg:pt-24 max-w-4xl m-auto text-center">
            <h1 className="text-white font-bold text-2xl md:text-4xl xl:text-5xl mb-2 md:mb-6 tracking-tight lg:leading-loose">
              Chat with sales engineering
            </h1>
            <p>
              We'll help you evaluate Inngest and show you how Inngest enables
              teams to ship more reliable code, faster.
            </p>
            <div className="flex place-content-center">
              <p className="mt-4 py-4 px-6 rounded-full bg-white/10 flex gap-2 items-center">
                👋&nbsp;&nbsp; Looking for support?{" "}
                <a href="/discord">Chat on Discord</a> or{" "}
                <a href={process.env.NEXT_PUBLIC_SUPPORT_URL}>
                  create a support ticket
                </a>{" "}
              </p>
            </div>
          </header>

          <div className="my-12 grid lg:grid-cols-2 gap-24">
            <form
              onSubmit={onSubmit}
              className="p-6 bg-indigo-900/10 text-indigo-100 flex flex-col items-start gap-4 rounded-lg border border-indigo-900/20"
            >
              <label className="w-full flex flex-col gap-2">
                Your name
                <input
                  type="text"
                  name="name"
                  onChange={(e) => setName(e.target.value)}
                  required
                  className="w-full p-3 bg-slate-1000/40 border border-indigo-900/50 outline-none rounded-md"
                />
              </label>
              <label className="w-full flex flex-col gap-2">
                Company email
                <input
                  type="email"
                  name="email"
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  className="w-full p-3 bg-slate-1000/40 border border-indigo-900/50 outline-none rounded-md"
                />
              </label>
              <label className="w-full flex flex-col gap-2">
                What can we help you with?
                <textarea
                  name="message"
                  required
                  onChange={(e) => setMessage(e.target.value)}
                  className="w-full min-h-[10rem] p-3 bg-slate-1000/40 border border-indigo-900/50 outline-none rounded-md"
                />
              </label>
              <label className="w-full flex flex-col gap-2">
                What's the size of your engineering team?
                <select
                  name="teamSize"
                  defaultValue=""
                  required
                  onChange={(e) => setTeamSize(e.target.value)}
                  className="px-3 py-3 bg-slate-1000/40 border border-indigo-900/50 outline-none rounded-md"
                >
                  <option value="" disabled>
                    Select an option
                  </option>
                  <option value="1">Just Me</option>
                  <option value="2-9">2-9</option>
                  <option value="10-30">10-20</option>
                  <option value="10-30">20-99</option>
                  <option value="10-30">100+</option>
                </select>
              </label>
              <div className="mt-4 w-full flex flex-row justify-items-end">
                <button
                  type="submit"
                  disabled={disabled}
                  className={`button group inline-flex items-center justify-center gap-0.5 rounded-full font-medium tracking-tight transition-all text-sm px-10 py-2.5 text-white bg-indigo-500 hover:bg-indigo-400 ${
                    disabled ? "opacity-50" : ""
                  }`}
                >
                  {buttonCopy}
                </button>
              </div>
            </form>

            <div className="mx-auto max-w-2xl">
              <Quote
                text="We were struggling with the complexities of managing our social media and e-commerce workflows. Thanks to Inngest, we were able to simplify our development process, speed up our time to market, and deliver a better customer experience. Inngest has become an essential tool in our tech stack."
                attribution={{
                  name: "Aivaras Tumas",
                  title: "CEO @ Ocoya",
                  avatar: "/assets/customers/ocoya-aivaras-tumas.png",
                }}
                variant="vertical"
                className="p-4 md:p-4"
              />
              <p className="mt-16 mb-8 text-lg font-semibold text-indigo-50/80">
                Trusted by
              </p>
              <div className="flex flex-row flex-wrap gap-8">
                <img
                  className="h-7"
                  src="/assets/customers/tripadvisor.svg"
                  alt="TripAdvisor"
                />
                <img
                  className="h-7"
                  src="/assets/customers/resend.svg"
                  alt="Resend"
                />
                <img
                  className="h-8"
                  src="/assets/customers/snaplet-dark.svg"
                  alt="Snaplet"
                />
              </div>
            </div>
          </div>
        </main>
      </Container>
      <Footer />
    </div>
  );
}
