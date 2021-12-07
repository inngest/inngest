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
              <p>Ingest webhooks, events from your API, UX events, or events via integrations &mdash; fully HA with zero infrastcture required</p>
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

          <p className="text-center">Plus, you can test easily using our local CLI, integrate easily with CI/CD tooling, leverage our prebuilt&nbsp;integrations, and get full schemas for every version of your events automatically.</p>
        </Content>

        <Content className="top-gradient">
          <header className="text-center">
            <h2>Build and iterate<br />without complexity</h2>
            <p>Easily create multi-step processes to automate anything, written as code or via a UI.  Then, have them run automatically every time events are received, on a schedule, or manually via your team.</p>
          </header>

          <BuildGrid className="text-center">
            <div>
              <img src="https://via.placeholder.com/350x150" />
              <h3>Build workflows rapidly</h3>
              <p>Create workflows using a fully typed config hosted in your own VCS, or use our UI to write the code for you.</p>
            </div>
            <div>
              <img src="https://via.placeholder.com/350x150" />
              <h3>Fully connected</h3>
              <p>Connect to all of your tools via integrations to common systems, with full API support and secrets built-in.</p>
            </div>
            <div>
              <img src="https://via.placeholder.com/350x150" />
              <h3>Manage & coordinate events</h3>
              <p>Automatically run workflows whenever events are received, or pause workflows until we receive new events (or don't).</p>
            </div>
          </BuildGrid>
        </Content>
        
        <Content>
          <SolveGrid>
            <header>
              <h2>Solve anything</h2>
              <p>Built to handle all complex behind-the-scenes flows engineers lose time building, with a library of examples to get started.</p>
            </header>

            <div>
              <h4>Customer journeys</h4>
              <small>eg. post signup flows</small>
              <p>Ensure new users are added to every system & campaign, with built-in integrations, version control, and handover to other teams</p>
            </div>

            <div>
              <h4>Real-time integrations</h4>
              <small>eg. billing &amp; support systems</small>
              <p>Respond to activity across all of your systems, such as running inference or auto-escalation with new support tickets, or handling payment failures</p>
            </div>

            <div>
              <h4>Scheduled jobs</h4>
              <small>eg. daily reports &amp; micro-batching</small>
              <p>Run workflows as scheduled jobs with zero infrastructure, config, and management, then see full logs &amp; history every time flows run</p>
            </div>

            <div>
              <h4>Sequenced flows</h4>
              <small>eg. churn & abandonment</small>
              <p>Coordinate between events or the lack of them, such as if a user doesn’t log in within 7 days after signup, or check out after adding to cart — all built in</p>
            </div>

            <div>
              <h4>Alerting</h4>
              <small>eg. security flows</small>
              <p>Create alerts any time events happen in your system with built-in integrations, such as new deploys or suspicious logins</p>
            </div>

            <div>
              <h4>Internal ops</h4>
              <small>eg. complex customer requests</small>
              <p>Build multi-step workflows that your entire team can manage and operate, such as refunding customers</p>
            </div>

          </SolveGrid>
        </Content>
      </div>

      <Content>
        <GetStarted>
          <h2>
            Ready to
            <br />
            get&nbsp;started?
          </h2>
          <div>
            <p>Inngest’s programmable serverless event platform allows you to get started building rapidly deployable, easily changeable workflows with zero infrastructure, that run whenever you need them to.</p>
            <p>Plus, you can create workflows and offload the operations of them to your wider team, using Inngest as an internal tool.</p>

            <div>
              <a href="https://app.inngest.com/register" class="button button--outline">Sign up →</a>

              <a href="https://www.inngest.com/docs">Explore documentation</a>
            </div>
          </div>
        </GetStarted>
      </Content>

      <Newsletter>
        <p><b>Bonus:  sign up to our newsletter?</b>  You’ve scrolled pretty far, and we didn’t really want to nag you earlier.  No pressure, and we’ll only send you fun & interesting things.  Like, say, news about open sourcing our execution platform!</p>

        {!submitted && (
          <form onSubmit={onSubmit} className={submitted ? "submitted" : ""}>
            <input
              type="email"
              onChange={(e) => setEmail(e.target.value)}
              value={email}
              placeholder="Your work email"
              required
            />
            <button type="submit" disabled={submitted}>
              Subscribe
            </button>
          </form>
        )}
        {submitted && (
          <p style={{ textAlign: "center", fontSize: 12, marginTop: "2rem" }}>
            You're added!  Only top-shelf stuff.  If not, yoink us from your inbox.
          </p>
        )}

      </Newsletter>

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
    border: 1px solid #ffffff19;
    padding: 30px 30px 30px 120px;
    border-radius: 5px;
  }

  p { opacity: .8 }

  & + p {
    opacity: .7;
    margin: 2rem auto;
    max-width: min(80vw, 850px);
  };
`

const BuildGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  grid-gap: 55px;

  img { margin: 0 10px 20px }
  p { opacity: .85; }
`;

const SolveGrid = styled.div`
  margin: 18vh 0;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);

  > div { padding: 50px 30px 30px; }

  header {
    grid-column: 1 / span 2;
    padding: 40px 50px;
  }

  h4 { margin: 0 }
  small { opacity: .5; font-size: 12px; }
  div p { font-size: 14px; margin-top: 1rem; }

  div:nth-of-type(1) { background: #0C1B46; }
  div:nth-of-type(2) { background: #282F68; }
  div:nth-of-type(3) { background: #193770; }
  div:nth-of-type(4) { background: #212B7A; }
  div:nth-of-type(5) { background: #1F3C74; }
  div:nth-of-type(6) { background: #263B63; }


`;

const GetStarted = styled.div`
  max-width: min(90vw, 1100px);
  margin: 18vh auto;
  display: grid;
  grid-template-columns: 1fr 2fr;
  grid-gap: 50px;

  p { font-size: 18px; opacity: .85; }
  p + div { margin: 3rem 0 0; font-size: 14px; }
  a.button { margin: 0 2rem 0 0 }
`;

const Newsletter = styled.div`
  width: min(90vw, 650px);
  margin: 0 auto 18vh;
  border: 1px solid #ffffff19;
  border-radius: 20px;
  padding: 30px;
  background: #00000233;
  box-shadow: 0 20px 80px rgba(0, 0,0, 0.7);

  p { opacity: .6; font-size: 14px; };

  form {
    display: flex;
    flex-direction: row;
    align-items: stretch;
    justify-content: center;
    margin: 2rem 0 0;
  }

  input {
    height: auto;
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
    font-size: 14px;
  }
  button {
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
    font-size: 14px;
  }
`;


