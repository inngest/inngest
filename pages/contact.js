import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";

import Workflow from "../shared/Icons/Workflow";
import Language from "../shared/Icons/Language";
import Lightning from "../shared/Icons/Lightning";
import Support from "../shared/Icons/Support";
import Audit from "../shared/Icons/Audit";
import { useState } from "react";

const CONTACT_KEY =
  "Z-ymc97Dae8u4HHybHknc4DGRb51u6NnTOUaW-qG71ah1ZqsJfRcI6SaHg5APWutNcnMcaN3oZrZky-VQxBIyw";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Contact Us",
        description: "Build event serverless event-driven systems in seconds",
      },
    },
  };
}

export default function Contact() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [msg, setMsg] = useState("");
  const [sent, isSent] = useState(false);

  const onSubmit = async (e) => {
    e.preventDefault();
    try {
      await Inngest.event(
        {
          name: "contact.form.sent",
          data: { email, name, message: msg },
          user: { email, name },
        },
        { key: CONTACT_KEY }
      );
    } catch (e) {
      console.warn("Message not sent");
    }
    isSent(true);
  };

  return (
    <>
      <Nav />

      <Hero>
        <h1>Contact us</h1>
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
  );
}

const Hero = styled.div`
  position: relative;
  z-index: 2;
  overflow: hidden;
  text-align: center;

  padding: 10vh 0;

  h1 + p {
    font-size: 22px;
    line-height: 1.45;
    opacity: 0.8;
  }
`;

const Inner = styled.div`
  form {
    border: 1px solid #ffffff19;
    border-radius: 7px;
    background: rgba(255, 255, 255, 0.03);
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
    padding: 0 2rem 2rem;
    max-width: 600px;
    margin: 0 auto;
    position: relative;

    > div:last-of-type {
      display: flex;
      justify-content: flex-end;
      width: 100%;
    }

    &:after {
      display: flex;
      align-items: center;
      justify-content: center;
      position: absolute;
      content: "Your message has been sent.  We'll be in touch shortly.";
      background: var(--bg-dark);
      top: 0;
      left: 0;
      bottom: 0;
      right: 0;
      pointer-events: none;
      transition: all 0.3s;
      opacity: 0;
    }

    &.sent:after {
      opacity: 1;
    }
  }

  h3 {
    margin: 2rem 0 0;
  }

  label,
  input,
  textarea {
    display: block;
  }
  label {
    margin: 2rem 0 0;
  }
  input,
  textarea {
    margin-top: 0.5rem;
    width: 100%;
  }
  textarea {
    min-height: 10rem;
  }

  button {
    margin: 2rem 0 0;
  }
`;
