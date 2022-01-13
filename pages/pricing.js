import { useEffect } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";
import { FinisherHeader } from "../shared/HeaderBG";

import Workflow from "../shared/Icons/Workflow";
import Language from "../shared/Icons/Language";
import Lightning from "../shared/Icons/Lightning";
import Plus from "../shared/Icons/Plus";
import Support from "../shared/Icons/Support";
import Audit from "../shared/Icons/Audit";

const gradient = (el, colors = ["#18435c", "#18435c", "#2f622f"]) => {
  new FinisherHeader(
    {
      count: 6,
      size: {
        min: 700,
        max: 900,
        pulse: 0,
      },
      speed: {
        x: {
          min: 0.1,
          max: 0.8,
        },
        y: {
          min: 0.1,
          max: 0.6,
        },
      },
      colors: {
        background: "#0f111e",
        particles: colors,
      },
      blending: "lighten",
      opacity: {
        center: 0.3,
        edge: 0,
      },
      shapes: ["c"],
    },
    el
  );
};

export default function Pricing() {
  useEffect(() => {
    gradient(document.querySelector(".pro"));
    gradient(document.querySelector(".advanced"), [
      "#893eb5",
      "#893eb5",
      "#b7672c",
    ]);
  }, []);

  const toggleFAQ = (e) => {
    e.currentTarget.classList.toggle("active");
  };

  return (
    <>
      <Head>
        <title>Inngest → programmable event platform pricing</title>
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

      <Nav />

      <Hero>
        <h1>Simple pricing. Powerful functionality.</h1>
        <p>
          Save weeks of development with our event-driven platform.
          <br />
          Get started in seconds for free, with plans that grow as you scale.
        </p>
      </Hero>

      <Content>
        <Free>
          <div>
            <h3>Community</h3>
            <p>
              Powerful enough to be useful, and <b>always free</b>. Perfect for
              getting started.
            </p>
            <ul>
              <li>
                <Workflow size="18" /> <b>5</b> &nbsp;workflows
              </li>
              <li>
                <Language size="18" /> <b>1,000</b> &nbsp;function runs/month
              </li>
              <li>
                <Lightning size="18" /> <b>Limited</b> &nbsp;resources &
                throughput
              </li>
            </ul>
          </div>
          <div>
            <a
              href="https://app.inngest.com/register"
              className="button button--outline"
            >
              Sign up for free →
            </a>
          </div>
        </Free>

        <Grid>
          <Plan>
            <PlanHeader className="pro">
              <div>
                <h3>Early adopter</h3>
                <p>
                  <b>$20/mo</b> &nbsp;
                  <small>discounted for being an early supporting member</small>
                </p>
                <a href="https://app.inngest.com/register" className="button">
                  Sign up as an early adopter →
                </a>
              </div>
            </PlanHeader>
            <ul>
              <li>
                <Workflow size="18" /> <b>50</b>&nbsp;workflows
              </li>
              <li>
                <Language size="18" /> <b>10,000</b>&nbsp;function runs/month
              </li>
              <li>
                <Lightning size="18" /> <b>Normal</b>&nbsp;resources &
                throughput
              </li>
              <li>
                <Audit size="18" /> <b>1 week</b>&nbsp;audit & log history
              </li>
            </ul>
          </Plan>

          <Plan>
            <PlanHeader className="advanced">
              <div>
                <h3>Advanced</h3>
                <p>
                  Powerful access for any scale. Contact us for more information
                </p>
                <a href="/contact" className="button">
                  Contact us →
                </a>
              </div>
            </PlanHeader>
            <ul>
              <li>
                <Workflow size="18" /> <b>Unlimited</b>&nbsp;workflows
              </li>
              <li>
                <Language size="18" /> <b>Unlimited</b>&nbsp;function runs/month
              </li>
              <li>
                <Lightning size="18" /> <b>Custom</b>&nbsp;resources &
                throughput
              </li>
              <li>
                <Audit size="18" /> <b>1 month</b>&nbsp;or more audit & log
                history
              </li>
              <li>
                <Plus size="18" /> <b>Add-ons</b>&nbsp;available
              </li>
              <li>
                <Support size="18" /> <b>Dedicated</b>&nbsp;support
              </li>
            </ul>
          </Plan>
        </Grid>

        <All>
          <p class="text-center">
            All plans include unlimited team members, uncapped events, scheduled
            worfklows, audit trails, API access, CD via our CLI, and first-class
            debugging.
          </p>
        </All>

        <hr />

        <h2 className="text-center">FAQs</h2>

        <FAQGrid>
          <div onClick={toggleFAQ}>
            <h3>What's a workflow?</h3>
            <p>
              Workflows are a sequence of serverless functions. Workflows can
              run automatically (each time an event is received), on a schedule,
              or manually via forms (in the case of internal tools, or
              one-offs).
            </p>
            <p>
              Workflows allow you to chain logic together, coordinate between
              events, and create complex user flows and operational logic for
              your company. A simple workflow might only run one single
              serverless function; this is still a workflow.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>Do you charge for events?</h3>
            <p>
              We're currently accepting all events, uncapped. Our "soft" limit
              is 1M events/mo across free and paid plans, and from 50M events/mo
              on custom plans. We don't throttle you, right now.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What resources are available?</h3>
            <p>
              Free accounts are limited to 128mb of ram and a maximum runtime of
              10 seconds per function. Paid accounts can utilize 1GB of ram and
              have a runime limit of 60 seconds per function. Advanced accounts
              can use up to 16GB of ram and can run functions for up to 6 hours;
              if you need this functionality{" "}
              <a href="/contact">get in touch with us</a>.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What if I need to run more functions?</h3>
            <p>
              That's okay! We'll alert you when you're nearing your cap and will
              throttle your workflows after hitting the limit. You can always
              purchase more capacity, but we won't apply overage fees unless you
              specify payment caps (coming soon). We dislike surprise costs as
              much as you.
            </p>
            <p>You can buy an extra 20,000 runs for $10.</p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What are add-ons?</h3>
            <p>
              Add-ons allow you to customise the funtionality of Inngest. For
              example, you can run the executor on-premise, increase the memory
              and CPU capablities of your workflows, increase the audit history
              length, and so on.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What is event coordination, and how can I use it?</h3>
            <p>
              Event coordination allows you to wait for specific events from
              within a workflow. For example, after creating a shopping cart you
              might want to wait for the "order created" event for up to 24
              hours. If this event is received, the person who created the
              shopping cart checked out. If the event isn't received within 24
              hours, you can run logic to handle churn.
            </p>
            <p>
              We allow you to coordinate between events and wait up to 6 months
              for new events. That's... quite some time.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>Are there limits to how complex I can make a workflow?</h3>
            <p>
              No, not really. You can chain hundreds of functions in parallel;
              we'll run it for you.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>
              Is there a limit to how often I can run scheduled workflows?
            </h3>
            <p>
              In the community edition, workflows are limited to 15 minute
              intervals and may be throttled for fair usage. Paid plans can run
              scheduled workflows every minute, and are not subject to
              throttling. Custom plans have their own dedicated pools for
              managing workflows.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>Can I run this on prem?</h3>
            <p>
              If you'd like to self-host Inngest or the executor platform [reach
              out to us with your needs](/contact).
            </p>
          </div>
        </FAQGrid>
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

const Plan = styled.div`
  border: 1px solid #ffffff19;
  border-radius: 7px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.03);
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);

  ul {
    list-style: none;
  }

  ul li {
    display: flex;
    align-items: center;
    margin: 1rem 0 0;
  }

  svg {
    margin: 0 0.75rem 0 0;
  }

  h3 {
    margin: 0;
    font-size: 1.5rem;
  }
`;

const Free = styled(Plan)`
  margin: 0 0 30px;
  display: grid;
  grid-template-columns: auto 200px;
  align-items: center;
  padding: 2rem;

  h3 + p {
    margin: 0.25rem 0 2rem;
  }

  ul {
    margin: 0;
    padding: 0;
    display: flex;
  }

  li {
    display: flex;
    align-items: center;
    margin: 0 3rem 0 0 !important;
  }

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
    grid-gap: 60px;
    padding-bottom: 3rem;

    ul {
      flex-direction: column;
    }
    li {
      margin: 0 !important;
    }
  }
`;

const PlanHeader = styled.div`
  padding: 3rem 2rem 3.5rem;
  margin: 0 0 ?rem;
  overflow: hidden;

  b {
    font-size: 1.1rem;
  }

  > div {
    z-index: 1;
    position: relative;
  }
`;

const Grid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-gap: 30px 30px;

  h3 + p {
    margin: 0.25rem 0 3.5rem;
  }

  small {
    opacity: 0.5;
    font-size: 100%;
  }

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
  }
`;

const All = styled.div`
  margin: 4vh 0;
  opacity: 0.8;
`;

const FAQGrid = styled.div`
  display: grid;

  div {
    cursor: pointer;

    h3 {
      font-size: 1.5rem;
      margin: 2rem 0;
    }
    p:last-of-type {
      margin: 0 0 3rem;
    }

    & + div {
      border-top: 1px solid rgba(255, 255, 255, 0.1);
    }

    & p {
      display: none;
    }
    &.active p {
      display: block;
    }
  }
`;
