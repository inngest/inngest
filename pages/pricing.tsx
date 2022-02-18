import { useEffect } from "react"
import styled from "@emotion/styled"
import Head from "next/head"
import Footer from "../shared/footer"
import Nav from "../shared/nav"
import Content from "../shared/content"
import Callout from "../shared/Callout"

import Block from "../shared/Block"
import IconList from "../shared/IconList"
import Button from "../shared/Button"

import Workflow from "../shared/Icons/Workflow"
import Language from "../shared/Icons/Language"
import Lightning from "../shared/Icons/Lightning"
import Plus from "../shared/Icons/Plus"
import Support from "../shared/Icons/Support"
import Audit from "../shared/Icons/Audit"

const PLANS = {
  free: {
    name: "Community",
    description: (
      <>
        Powerful enough to be useful, and <strong>always free</strong>. Perfect
        for getting started.
      </>
    ),
    features: [
      {
        icon: Workflow,
        quantity: "5",
        text: "workflows",
      },
      {
        icon: Language,
        quantity: "1,000",
        text: "function runs/month",
      },
      {
        icon: Lightning,
        quantity: "Limited",
        text: "resources & throughput",
      },
    ],
  },
  pro: {
    name: "Early Adopter",
    description: (
      <>
        <strong>$20/mo</strong> &nbsp;
        <small>discounted for being an early supporting member</small>
      </>
    ),
    features: [
      {
        icon: Workflow,
        quantity: "50",
        text: "workflows",
      },
      {
        icon: Language,
        quantity: "10,000",
        text: "function runs/month",
      },
      {
        icon: Lightning,
        quantity: "Generous",
        text: "resources & throughput",
      },
      {
        icon: Audit,
        quantity: "1 week",
        text: "audit & log history",
      },
    ],
  },
  advanced: {
    name: "Advanced",
    description: <>Powerful access for any scale.</>,
    features: [
      {
        icon: Workflow,
        quantity: "Unlimited",
        text: "workflows",
      },
      {
        icon: Language,
        quantity: "Unlimited",
        text: "function runs/month",
      },
      {
        icon: Lightning,
        quantity: "Custom",
        text: "resources & throughput",
      },
      {
        icon: Audit,
        quantity: "1 month",
        text: "or more audit & log history",
      },
      {
        icon: Plus,
        quantity: "Add-ons",
        text: "available",
      },
      {
        icon: Support,
        quantity: "Add-ons",
        text: "available",
      },
    ],
  },
}

export default function Pricing() {
  const toggleFAQ = (e) => {
    e.currentTarget.classList.toggle("active")
  }

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
        <h1>
          Simple pricing.
          <br />
          Powerful functionality.
        </h1>
        <p>
          Save weeks of development with our event-driven platform.
          <br />
          Get started in seconds for free, with plans that grow as you scale.
        </p>
      </Hero>

      <Content>
        <Block color="primary">
          <PlanHeader flexDirection="row">
            <div>
              <h3>{PLANS.free.name}</h3>
              <p>{PLANS.free.description}</p>
            </div>
            <div>
              <Button kind="outlineHighContrast" href="/sign-up">
                Sign up for free →
              </Button>
            </div>
          </PlanHeader>
          <IconList collapseWidth={800} items={PLANS.free.features} />
        </Block>

        <Grid>
          <Block>
            <PlanHeader flexDirection="column">
              <h3>{PLANS.pro.name}</h3>
              <p>{PLANS.pro.description}</p>
            </PlanHeader>
            <div style={{ marginBottom: "2rem" }}>
              <Button kind="primary" href="/sign-up">
                Sign up as an early adopter →
              </Button>
            </div>
            <IconList direction="vertical" items={PLANS.pro.features} />
          </Block>

          <Block>
            <PlanHeader flexDirection="column">
              <h3>{PLANS.advanced.name}</h3>
              <p>{PLANS.advanced.description}</p>
            </PlanHeader>
            <div style={{ marginBottom: "2rem" }}>
              <Button kind="primary" href="/contact">
                Contact us →
              </Button>
            </div>
            <IconList direction="vertical" items={PLANS.advanced.features} />
          </Block>
        </Grid>

        <AllPlansInfo>
          All plans include unlimited team members, uncapped events, scheduled
          workflows, audit trails, API access, CD via our CLI, and first-class
          debugging.
        </AllPlansInfo>

        <h2 style={{ margin: "2rem 0" }}>FAQs</h2>

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

        <Callout />
      </Content>

      <div style={{ marginTop: 100 }}>
        <Footer />
      </div>
    </>
  )
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
  p {
    font-family: var(--font);
  }
`

const PlanHeader = styled.div<{
  flexDirection: "row" | "column"
}>`
  display: flex;
  flex-direction: ${(props) => props.flexDirection || "row"};
  justify-content: space-between;
  margin-bottom: 2rem;

  @media (max-width: 800px) {
    flex-direction: column;
  }

  p {
    margin: 1rem 0;
  }
`

const Grid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-gap: 2rem 2rem;
  margin: 2rem 0;

  small {
    opacity: 0.5;
    font-size: 100%;
  }

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
  }
`

const AllPlansInfo = styled.p`
  margin: 2rem auto 4rem;
  max-width: 40rem;
  line-height: 1.5rem;
  text-align: center;
`

const FAQGrid = styled.div`
  display: grid;
  margin-bottom: 3rem;

  div {
    cursor: pointer;

    h3 {
      margin: 1rem 0;
      font-family: var(--font);
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
`
