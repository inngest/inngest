import styled from "@emotion/styled";
import Head from "next/head";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import theme from "react-syntax-highlighter/dist/cjs/styles/prism/dracula";

export default function About() {
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
      <Nav>
        <div>
          <img src="/logo-blue.svg" alt="Inngest logo" />
        </div>
        <div>
          <a href="https://docs.inngest.com/docs/intro" target="_blank">
            Documentation
          </a>
          <a href="https://app.inngest.com/login">Sign in</a>

          <a
            href="https://calendly.com/inngest-thb/30min"
            className="button"
            rel="nofollow"
            target="_blank"
          >
            Request a free demo
          </a>
        </div>
      </Nav>

      <Hero className="text-center">
        <h1>Trigger low-code logic from events</h1>
        <p>
          Companies use Inngest to build real time, event driven workflows
          in&nbsp;minutes. <br />
          It's <strong>made for builders</strong>,{" "}
          <strong>designed for operators</strong>.
        </p>

        <a
          href="https://calendly.com/inngest-thb/30min"
          className="button"
          rel="nofollow"
          target="_blank"
        >
          Request a free demo
        </a>

        <div>
          <img
            src="/wflow.png"
            alt="An example cloud kitchen workflow when paying via Venmo"
          />
        </div>
      </Hero>

      <Content>
        <Tagline>
          <div>
            <Check />
            <p>Define workflows as code or&nbsp;via&nbsp;a&nbsp;UI</p>
          </div>
          <div>
            <Check />
            <p>Utilize pre-built integrations</p>
          </div>
          <div>
            <Check />
            <p>Run your own code, using&nbsp;any&nbsp;language</p>
          </div>
          <div>
            <Check />
            <p>Full user transparency &amp; audit&nbsp;trails</p>
          </div>
          <div>
            <Check />
            <p>Complete workflow version histories</p>
          </div>
        </Tagline>
      </Content>

      <Introducing>
        <Content>
          <h5>Introducing Inngest</h5>

          <p>
            Inngest is an <strong>automation platform</strong> which{" "}
            <strong>runs workflows on a schedule</strong> or{" "}
            <strong>in real-time after events happen</strong>.
            Design&nbsp;complex operational flows and run any code - including
            pre-built integrations or your own code - with
            zero&nbsp;infrastructure and&nbsp;maintenance.
          </p>

          <IntroGrid>
            <div>
              <h2>Workflow automation</h2>
              <p>
                Build, manage, and operate your product and ops flows
                end-to-end. Complete with out-of-the-box integrations for rapid
                development, and the ability to run your own serverless code for
                full&nbsp;flexibility
              </p>
            </div>
            <div>
              <h2>Change management</h2>
              <p>
                Version every workflow complete with history, schedule workflows
                to go live, and handle workflow approvals within your account -
                it’s everything you need for a fully compliant&nbsp;solution
              </p>
            </div>
            <div>
              <h2>Transparency &amp; debugging</h2>
              <p>
                Drill down into every workflow run, including which users ran
                through which versions of a workflow and each
                workflow’s&nbsp;logs.
              </p>
            </div>
          </IntroGrid>
        </Content>
      </Introducing>

      <Content>
        <Callout className="text-center">
          <div>
            <span>
              25<small>✕</small>
            </span>
            <strong>faster implementation</strong>
            <span>using our platform and integrations</span>
          </div>

          <div>
            <span>
              20<small>✕</small>
            </span>
            <strong>faster debugging &amp; editing</strong>
            <span>with our insights, logs, and editor</span>
          </div>

          <div>
            <span>
              15<small>✕</small>
            </span>
            <strong>more cost effective</strong>
            <span>than deploying &amp; managing yourself</span>
          </div>
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

      <Footer>
        <Content>
          <Links>
            <div>
              <img src="/logo-white.svg" alt="Inngest logo" height="30" />
              <small>© 2021 Inngest Inc</small>
            </div>
            <div>
              <strong>Inngest</strong>
              <a href="https://www.inngest.com">Product</a>
              <a href="https://docs.inngest.com">Documentation</a>
            </div>
            <div>
              <strong>Community</strong>
              <a href="https://discord.gg/EuesV2ZSnX">Discord</a>
              <a href="https://twitter.com/inngest">Twitter</a>
            </div>
          </Links>
        </Content>
      </Footer>
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
        <h3>Implement any realtime logic you can dream of, in minutes</h3>

        <div>
          <HIWGrid>
            <li>
              <strong>Send us events</strong>
              <p>Send us events via the API, SDK, webhooks, or integrations.</p>
            </li>
            <li>
              <strong>Configure your workflows</strong>
              <p>
                Create your workflows via the low-code UI or by writing code
                directly.
              </p>
            </li>
            <li>
              <strong>Run workflows in real time</strong>
              <p>
                Workflows automatically run in real time on each event or on a
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

const Nav = styled(Content)`
  height: 70px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  font-size: 0.9rem;

  img {
    max-height: 40px;
  }

  a:not(.button) {
    display: inline-block;
    padding: 12px 20px 11px;
    transition: all 0.2s;
    text-decoration: none;
  }
  a:not(.button):hover {
    background: #2f6d9d11;
  }

  a + a {
    margin-left: 5px;
  }

  a + a.button {
    margin-left: 20px;
  }

  @media only screen and (max-width: 800px) {
    div:last-of-type {
      display: none;
    }
  }
`;

const Hero = styled(Content)`
  font-size: 1.3125rem;
  padding: 80px 0 0;
  position: relative;

  .button {
    display: inline-block;
    font-size: 1rem;
    margin-top: 40px;
    width: auto;
    height: auto;
  }

  > div {
    box-shadow: 0 10px 50px rgba(0, 0, 0, 0.1);
    background: #fffefc;
    width: 100%;
    max-width: 100%;
    height: 480px;
    max-height: 500px;
    margin: 90px 0 0;
    position: relative;
    z-index: 2;
    overflow: hidden;
  }

  @media only screen and (max-width: 800px) {
    h1 {
      line-height: 1;
      font-size: 50px;
      margin: 40px 0;
    }

    > div {
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
  background: linear-gradient(
    180deg,
    rgba(243, 245, 245, 1) 20%,
    rgba(249, 251, 254, 1) 100%
  );
  padding: 450px 40px 180px 40px;
  margin-top: -400px;

  h5 + p {
    font-size: 1.3125rem;
  }

  @media only screen and (max-width: 800px) {
    margin-top: 20px;
    padding-top: 40px;
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
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-gap: 40px;

  background: #fdfbf6;
  padding: 40px;
  box-shadow: 0 10px 50px rgba(0, 0, 0, 0.1);

  strong,
  span {
    display: block;
    margin: 4px 0;
  }

  small {
    font-size: 1.5rem;
  }

  span:first-of-type {
    font-size: 2.6rem;
    margin: 0 0 6px;
  }

  span:last-of-type {
    color: #737885;
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

const Footer = styled.div`
  margin-top: 100px;
  background: #222631;
  color: #fff;
  padding: 40px 0;
  font-size: 0.9rem;
`;

const Links = styled.div`
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  font-size: 14px;
  line-height: 1.8;
  opacity: 0.7;

  img {
    display: block;
    margin: 0 0 4px;
  }

  strong {
    display: block;
    margin-bottom: 10px;
  }

  a {
    display: block;
    color: #fff;
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
