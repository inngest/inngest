import { useEffect, useState } from "react";

import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";

const CONTACT_KEY =
  "Z-ymc97Dae8u4HHybHknc4DGRb51u6NnTOUaW-qG71ah1ZqsJfRcI6SaHg5APWutNcnMcaN3oZrZky-VQxBIyw";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Contact Us",
      },
      designVersion: "2",
    },
  };
}

export default function Contact() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [message, setMessage] = useState("");
  const [disabled, setDisabled] = useState<boolean>(false);
  const [buttonCopy, setButtonCopy] = useState("Send");

  const onSubmit = async (e) => {
    e.preventDefault();
    setDisabled(true);
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
          data: { email, name, message, ref },
          user: { email, name },
          v: "2023-04-07.1",
        },
        { key: CONTACT_KEY }
      );
    } catch (e) {
      console.warn("Message not sent");
      setButtonCopy("Message not sent");
    }
    setDisabled(false);
    setButtonCopy("Sent!");
  };

  return (
    <div className="font-sans text-slate-200">
      <Header />
      <Container>
        <main className="m-auto max-w-[65ch] pt-16 pb-8">
          <header className="pt-12 lg:pt-24 max-w-[65ch] m-auto">
            <h1 className="text-white font-medium text-2xl md:text-4xl xl:text-5xl mb-2 md:mb-4 tracking-tighter lg:leading-loose">
              Get in touch
            </h1>
            <p>
              Have a question about the product or want a demo? <br />
              Complete the form below or{" "}
              <a href="mailto:hello@inngest.com">email us</a>.
            </p>
          </header>

          <form
            onSubmit={onSubmit}
            className="my-12 p-6 bg-indigo-900/20 text-indigo-100 flex flex-col items-start gap-4 rounded-lg border border-indigo-900/50"
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
                className="w-full p-3 bg-slate-1000/40 border border-indigo-900/50 outline-none rounded-md"
              />
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
        </main>
      </Container>
      <Footer />
    </div>
  );
  /*let x = (
    <>

      <Hero className="pt-16 pb-12">
        <h1 className="pb-4">Contact us</h1>
        <p>
          How can we help you?
          <br /> Reach out to us by live chat,{" "}
          <a href="mailto:hello@inngest.com">email</a>, or the form below for
          help.
        </p>
      </Hero>

      <Content>
        <Inner>
          <form onSubmit={onSubmit} className={sent ? "sent" : ""}>
            <label>
              Your name
              <input
                type="text"
                name="name"
                onChange={(e) => setName(e.target.value)}
                placeholder="Your name, please"
                required
              />
            </label>
            <label>
              Your email
              <input
                type="email"
                name="email"
                onChange={(e) => setEmail(e.target.value)}
                placeholder="Your email address too"
                required
              />
            </label>
            <label>
              Your message to the team
              <textarea
                name="message"
                required
                onChange={(e) => setMsg(e.target.value)}
                placeholder="Your message.  It'll go straight to the team, and you'll hear back within the day."
              />
            </label>
            <div>
              <button type="submit">Send</button>
            </div>
          </form>
        </Inner>
      </Content>

      <div style={{ marginTop: 100 }}>
        <Footer />
      </div>
    </>
  );*/
}
