import { useState } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";

// TODO: move these into env vars
// prod key
export const INGEST_KEY =
  "BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ";

// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

export default function Home() {
  const [submitted, setSubmitted] = useState(false);
  const [email, setEmail] = useState("");

  const onSubmit = (e) => {
    e.preventDefault();
    const Inngest = globalThis.Inngest;
    if (!Inngest) {
      console.warn("Inngest not found");
      return;
    }
    Inngest.init(INGEST_KEY);
    Inngest.event({
      name: "marketing.signup",
      data: { email },
      user: { email },
    });
    setEmail("");
    setSubmitted(true);
  };

  return (
    <>
      <Head>
        <title>
          Inngest → build serverless event-driven systems in seconds
        </title>
        <link rel="icon" href="/favicon.png" />
        <meta property="og:title" content="Inngest" />
        <meta property="og:url" content="https://www.inngest.com" />
        <meta property="og:image" content="/logo.svg" />
        <meta
          property="og:description"
          content="Build event serverless event-driven systems in seconds"
        />
        <script src="/inngest-sdk.js"></script>
        <script
          defer
          src="https://static.cloudflareinsights.com/beacon.min.js"
          data-cf-beacon='{"token": "e2fa9f28c34844e4a0d29351b8730579"}'
        ></script>
      </Head>

      <Nav dark />

      <Hero>
        <Content className="grid">

          <div>
            <h1>Build serverless event&nbsp;driven systems, <i>in seconds</i></h1>
            <p>A flexible platform which manages all of your events and runs serverless workloads in real time &mdash; automating anything you need, with zero&nbsp;config&nbsp;or&nbsp;infra.</p>

            <div class="cta">
              <a hrefName="https://app.inngest.com/register">Join the preview</a>
              or <a href="/contact">speak with us</a>
            </div>
          </div>

          <div style={{ background: "#fff", opacity: 0.2 }}>
            SCREENSHOT
          </div>

        </Content>
      </Hero>

      <div>
        <Content>
          <header className="text-center">
            <h2>
              Made for developers.<br />
              Designed for teams.
            </h2>
            <p>Our platform is crafted to help you build processes faster, and designed so that your entire team can understand&nbsp;and&nbsp;operate&nbsp;them.</p>
          </header>

          <FeatureGrid>

            <div>
              <h3>Track any event</h3>
              <p>Ingest webhooks, events from your API, UX events, or events via integrations with zero infrastcture required</p>
            </div>

            <div>
              <h3>Audit trails</h3>
              <p>See which users are responsible for every event and action in your system, with infinite retention</p>
            </div>

            <div>
              <h3>Any runtime, any language</h3>
              <p>Hook into your existing infra via AWS Lambda and Cloudflare Workers. Or, run any code within your workloads, in any language</p>
            </div>

            <div>
              <h3>Manual approvals &amp; coordination</h3>
              <p>Automate complex flows with built-in manual approvals, and built-in event coordination with timeouts</p>
            </div>

            <div>
              <h3>Version control</h3>
              <p>Every workload you build is fully versioned, allowing you to schedule releases and roll back quickly</p>
            </div>

            <div>
              <h3>Logging, debugging, & retries</h3>
              <p>First-class support for logging, a step-over debugger, built-in retries, and error management out of the box</p>
            </div>

          </FeatureGrid>

          <p class="text-center">Plus, you can test easily using our local CLI, integrate easily with CI/CD tooling, leverage our prebuilt&nbsp;integrations, and get full schemas for every version of your events automatically.</p>
        </Content>
      </div>

      {/*
      <Signup>
        <form onSubmit={onSubmit} className={submitted ? "submitted" : ""}>
          <input
            type="email"
            onChange={(e) => setEmail(e.target.value)}
            value={email}
            placeholder="Your work email"
            required
          />
          <button type="submit" disabled={submitted}>
            Sign up for updates
          </button>
        </form>
        {submitted && (
          <p style={{ textAlign: "center", fontSize: 12 }}>
            You're on the list and will receive an invite soon!
          </p>
        )}
      </Signup>
      */}

      <div style={{ marginTop: 100 }}>
        <Footer />
      </div>
    </>
  );
}

const Hero = styled.div`
  padding: 10vh 0;
  border-bottom: 4px solid #ffffffdd;

  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    > div:first-of-type { z-index: 1; }
    > div:last-of-type { z-index: 0 }
  }

  h1 { width: 130%; }
  h1 + p { font-size: 22px; line-height: 1.45; }

  .cta {
    margin: 60px 0 0;

    a:first-of-type {
      display: inline-block;
      border: 1px solid #fff;
      border-radius: 3px;
      color: #fff;
      line-height: 1;
      text-decoration: none;
      padding: 12px 18px 14px;
      margin: 0 20px 0 0;
    }

    a:last-of-type {
      display: inline-block;
      margin-left: 12px;
    }
  }
`;

const FeatureGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-gap: 30px 30px;

  > div {
    border: 1px solid #ffffff22;
    padding: 30px 30px 30px 120px;
    border-radius: 5px;
  }

  p { opacity: .8 }

  & + p {
    opacity: .7;
    margin: 2rem auto;
    max-width: 80vw;
  };
`


const Signup = styled.div`
`;

