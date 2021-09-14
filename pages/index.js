import { useState } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import theme from "react-syntax-highlighter/dist/cjs/styles/prism/dracula";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Workflow from "../shared/workflow";
import Check from "../shared/icons/check";

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
      data: {
        email,
      },
      user: {
        email,
      },
    });

    setEmail("");
    setSubmitted(true);
  };

  return (
    <>
      <Head>
        <title>
          Inngest → serverless event-driven & scheduled workflow automation
          platform for developers & operators
        </title>
        <link rel="icon" href="/favicon.png" />
        <meta property="og:title" content="Inngest" />
        <meta property="og:url" content="https://www.inngest.com" />
        <meta property="og:image" content="/logo.svg" />
        <meta
          property="og:description"
          content="Build, run, operate, and analyze your workflows in minutes."
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
            <h1>Trigger low-code logic from events</h1>
            <p>
              <strong>
                Build real time, event driven workflows in&nbsp;minutes,
                with&nbsp;or&nbsp;without engineering&nbsp;teams.
                <br />
              </strong>{" "}
              It's made for builders, designed for operators.
            </p>

          <a
            href="https://calendly.com/inngest-thb/30min"
            className="button"
            rel="nofollow"
            target="_blank"
          >
            See how it works
          </a>

          {/*
          <div className="img">
            <img
              src="/wflow.png"
              alt="An example cloud kitchen workflow when paying via Venmo"
            />
          </div>
          */}
          </div>
          <div className="workflow">
            <Workflow />
          </div>
        </Content>

      </Hero>
      

      <Content>
        <Tagline>
          <div>
            <Check />
            <p>Define workflows in code or in&nbsp;a&nbsp;visual&nbsp;UI</p>
          </div>
          <div>
            <Check />
            <p>Integrate with your existing&nbsp;tools</p>
          </div>
          <div>
            <Check />
            <p>Switch integrations in seconds,&nbsp;with&nbsp;zero&nbsp;code</p>
          </div>
          <div>
            <Check />
            <p>Run custom code, in any&nbsp;language</p>
          </div>
          <div>
            <Check />
            <p>Complete workflow version&nbsp;histories</p>
          </div>
        </Tagline>
      </Content>

      <Introducing>
        <Content class="text-center">
          <HighLevel>
            <h2>Automation running in minutes</h2>
            <p>
              Inngest aggregates events from your internal &amp; external
              systems and runs any code when things happen in your business.
            </p>
            <a href="/product" className="button button--outline">
              Learn more about the platform →

            </a>
          </HighLevel>
        </Content>

        <Content className="grid">
          <div>
            <h5>Introducing Inngest</h5>
            <h2>

              React to everything,
              <br />
              with zero code
            </h2>

            <p>
              Build complex automations via drag and drop - with or without
              engineering. Instantly test and deploy new workflow versions,
              with a full version history and changelog built-in.
            </p>
            <p>
              <strong>
                Inngest lets you build event-driven logic while your engineering
                team focuses on the core product.
              </strong>
            </p>

          </div>
          <div></div>

          <div>
            <h2>Integrate anything, instantly</h2>

            <p>
              <strong>Inngest lets you build faster.</strong> We let you
              integrate with any API, no code required. When your requirements
              change, it only takes a few seconds to set up and swap your next
              integration.
            </p>
          </div>

          <div></div>

          <div>
            <h2>Instant, reliable, and&nbsp;flexible</h2>

            <p>
              Your workflows run in real-time, with full logging, audit trails,
              and retries out of the box.
            </p>
            <p>
              Need something more complex? Run your own custom code as part of a
              workflow,{" "}
              <strong>without worrying about servers or server code</strong>.
            </p>
          </div>

          <div></div>
        </Content>
      </Introducing>

      <Content>
        <Callout className="text-center">
          <h5>Integrates with your existing tools</h5>
          <IntegrationsIcons>
            <img src="integrations/salesforce.png" alt="Salesforce" />
            <img
              style={{ height: "30px" }}
              src="integrations/jira.png"
              alt="Atlassian Jira"
            />
            <img src="integrations/clickup.png" alt="Clickup" />
            <img src="integrations/stripe.png" alt="Stripe" />
            <img src="integrations/onesignal.png" alt="One Signal" />
            <img src="integrations/slack.png" alt="Slack" />
            <img src="integrations/twilio.png" alt="Twilio" />
          </IntegrationsIcons>

          <small>and many more.</small>
        </Callout>
      </Content>

      <HowItWorks />

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

      <div style={{ marginTop: 100 }}>
        <Footer />
      </div>
    </>
  );
}

const send = `// Send us events with a single call.  Libraries provided for
// the browser, node, python, and Go.
Inngest.event({
  name: "signup.new",
  data: {
    email: "some@new.example.com",
    plan: "enterprise",
    sector: "fintech",
  },
  user: {
    email: "some@new.example.com",
    first_name: "Jazmine",
    last_name: "Doe",
  }
});`;

const HowItWorks = () => {
  return (
    <HIW>
      <Content>
        <h5>How it works</h5>
        <h3>Implement any real-time logic you can dream of, in minutes.</h3>

        <div>
          <HIWGrid>
            <li>
              <strong>Send us events</strong>
              <p>
                Send us events through the API, SDK, webhooks, or integrations.
              </p>
            </li>
            <li>
              <strong>Configure your workflows</strong>
              <p>
                Create your workflows using the drag-and-drop UI or by writing
                code directly.
              </p>
            </li>
            <li>
              <strong>Run workflows in real time</strong>
              <p>
                Workflows automatically run in real-time on each event or on a
                schedule.
              </p>
            </li>
            <li>
              <strong>Manage your automations</strong>
              <p>
                Easily manage your workflows, with full version histories and
                visibility into which users run through which versions.
              </p>
            </li>
          </HIWGrid>

          <div>
            <Code>
              <SyntaxHighlighter language="javascript" style={theme}>
                {send}
              </SyntaxHighlighter>
            </Code>
          </div>
        </div>
      </Content>
    </HIW>
  );
};

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;

  @media only screen and (max-width: 800px) {
    padding: 0 20px;
  }
`;

const Hero = styled.div`
  font-size: 1.3125rem;
  padding: 100px 0 0;
  position: relative;
  color: #fff;

  background: linear-gradient(90deg, var(--blue-left), var(--blue-right));
  box-shadow: inset 0 0 0 20px #fff;

  min-height: 400px;

  &:before {
    content: "";
    display: block;
    background: linear-gradient(90deg, var(--blue-left), var(--blue-right));
    height: 20px;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    border-left: 20px solid #fff;
    border-right: 20px solid #fff;
  }

  > div {
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-gap: 80px;
  }

  h1 {
    color: transparent;
    background: linear-gradient(90deg, #fff, #f5f5f5);
    color: #fff;
    background-clip: text;
  }

  .button {
    display: inline-block;
    font-size: 1rem;
    margin-top: 40px;
    width: auto;
    height: auto;
  }

  .img {
    box-shadow: 0 10px 50px rgba(0, 0, 0, 0.1);
    background: #fffefc;
    width: 100%;
    max-width: 100%;
    height: 480px;
    max-height: 500px;
    margin: 90px 0 0;
    position: relative;
    z-index: 1;
    overflow: hidden;
  }

  .workflow {
    max-width: 100%;
    height: 480px;
    max-height: 500px;
  }

  @media only screen and (max-width: 800px) {
    h1 {
      line-height: 1;
      font-size: 50px;
      margin: 40px 0;
    }

    .img {
      display: none;
    }
  }
`;

const Tagline = styled.div`
  padding: 40px 40px 0;
  font-size: 13px;
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  grid-gap: 40px;
  text-align: center;
  position: relative;
  z-index: 2;

  > div {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  p {
    margin: 4px 0;
    line-height: 1.3;
    opacity: 0.65;
  }

  @media only screen and (max-width: 800px) {
    grid-template-columns: 1fr;
    grid-template-rows: auto;
    grid-gap: 7px;

    > div {
      display: flex;
      flex-direction: row;
      text-align: left;
    }

    p {
      margin-left: 10px;
    }
  }
`;

const Introducing = styled.div`
  box-shadow: inset 0 0 0 20px #fff;
  background-repeat: repeat;
  background-image: linear-gradient(
    180deg,
    rgba(243, 245, 245, 1) 20%,
    rgba(249, 251, 254, 1) 100%
  );
  padding: 180px 40px 180px 40px;
  margin-top: -120px;
  position: relative;

  h5 + p {
    font-size: 1.3125rem;
  }

  @media only screen and (max-width: 800px) {
    margin-top: 20px;
    padding-top: 40px;
  }

  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-gap: 80px;
  }
`;

const HighLevel = styled.div`
  margin: 3rem 0 8rem;
  h2 {
    margin: 0 0 1.5rem;
  }

  p {
    font-size: 1.3rem;
    max-width: 40%;
    margin: 0 auto 3rem;
    line-height: 1.35;
    color: var(--dark-grey);
  }
`;

const IntroGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-gap: 100px;
  padding: 30px 0 0;

  @media only screen and (max-width: 800px) {
    grid-template-columns: 1fr;
    grid-gap: 20px;
  }
`;

const Callout = styled.div`
  max-width: 80%;
  margin: -115px auto 50px auto;

  background: #fdfbf6;
  padding: 40px;
  box-shadow: 0 10px 50px rgba(0, 0, 0, 0.1);

  display: flex;
  flex-flow: column;
  align-items: center;
  position: relative;
  z- > h5 {
    align-self: center;
  }

  @media only screen and (max-width: 800px) {
    grid-template-columns: 1fr;
    grid-gap: 20px;
  }
`;

const HIW = styled.div`
  > div > div {
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-gap: 80px;
  }

  @media only screen and (max-width: 800px) {
    > div > div {
      grid-template-columns: 1fr;
    }
  }
`;

const HIWGrid = styled.ol`
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: auto;
  grid-gap: 50px;
  padding: 0;

  li,
  strong,
  p {
    margin: 0;
  }
  strong {
    display: block;
    margin: 0 0 10px;
  }

  @media only screen and (max-width: 800px) {
    grid-template-columns: 1fr;
    padding: 0 20px;
  }
`;

const Signup = styled(Content)`
  padding: 10px 0 0;
  form {
    display: flex;
    align-items: stretch;
    justify-content: center;
    margin: 60px 0 0;
  }

  form,
  form * {
    font-size: 1rem;
  }

  button {
    width: 200px;
    line-height: 1;
  }

  @media only screen and (max-width: 800px) {
    form {
      flex-direction: column;
      align-items: stretch;
      justify-content: center;
    }
    button,
    input {
      width: auto;
    }
    input {
      padding: 24px 16px;
    }
  }
`;

const Code = styled.div`
  font-size: 14px;
  margin-top: -10px;
  max-width: 90vw;

  @media only screen and (max-width: 800px) {
    margin-top: -40px;
  }
`;

const IntegrationsIcons = styled.div`
  display: flex;
  flex-flow: row;
  justify-content: center;
  align-items: center;

  flex-wrap: wrap;
  margin-top: 50px;
  column-gap: 80px;

  > img {
    height: 50px;
    margin-bottom: 40px;
  }
`;

const Box = styled.div`
  box-sizing: border-box;
  border: 1px solid #e8e8e699;
  border-radius: 5px;
  background: #fff;
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.05);
  padding: 1.5rem;
`

const Study = styled(Box)`
  margin: 4rem 2rem;
`
