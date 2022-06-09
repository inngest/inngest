import { useEffect } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";
import Callout from "../shared/Callout";

import Block from "../shared/Block";
import IconList from "../shared/IconList";
import Button from "../shared/Button";

import Workflow from "../shared/Icons/Workflow";
import Language from "../shared/Icons/Language";
import ArrowCaretCircleRight from "../shared/Icons/ArrowCaretCircleRight";
import Plus from "../shared/Icons/Plus";
import ListCheck from "../shared/Icons/ListCheck";
import UserVoice from "../shared/Icons/UserVoice";

import Functions from "../shared/Icons/Play";
import UsersGroup from "../shared/Icons/UsersGroup";

const PLANS = [
  {
    name: "Community",
    cost: "free",
    description: (
      <>
        Powerful enough to be useful, and <strong>always free</strong>. Perfect
        for getting started.
      </>
    ),
    cta: {
      href: "/sign-up?ref=pricing-free",
      text: "Sign up for free →",
    },
    features: [
      {
        icon: ArrowCaretCircleRight,
        quantity: "3",
        text: "functions",
      },
      {
        icon: Language,
        quantity: "1,000",
        text: "function runs/month",
      },
      {
        icon: UsersGroup,
        quantity: "3",
        text: "seats",
      },
      {
        icon: ListCheck,
        quantity: "1 week",
        text: "log retention",
      },
    ],
  },
  {
    name: "Startup",
    cost: "$50/mo",
    description: <>Get to market fast.</>,
    cta: {
      href: "/sign-up?ref=pricing-startup",
      text: "Start building →",
    },
    features: [
      {
        icon: ArrowCaretCircleRight,
        quantity: "25",
        text: "functions",
      },
      {
        icon: Language,
        quantity: "25,000",
        text: "function runs/month",
      },
      {
        icon: UsersGroup,
        quantity: "5",
        text: "seats",
      },
      {
        icon: ListCheck,
        quantity: "1 month",
        text: "log retention",
      },
    ],
  },
  {
    name: "Team",
    cost: "$200/mo",
    description: <>More room to grow.</>,
    cta: {
      href: "/sign-up?ref=pricing-team",
      text: "Start building →",
    },
    features: [
      {
        icon: ArrowCaretCircleRight,
        quantity: "100",
        text: "functions",
      },
      {
        icon: Language,
        quantity: "100,000",
        text: "function runs/month",
      },
      {
        icon: UsersGroup,
        quantity: "20",
        text: "seats",
      },
      {
        icon: ListCheck,
        quantity: "3 month",
        text: "log retention",
      },
      {
        icon: UserVoice,
        quantity: "Dedicated",
        text: "support",
      },
    ],
  },
  {
    name: "Custom",
    cost: "Flexible pricing",
    description: <>Powerful access for any scale.</>,
    cta: {
      href: "/contact?ref=pricing-advanced",
      text: "Get in touch",
    },
    features: [
      {
        icon: ArrowCaretCircleRight,
        quantity: "Custom",
        text: "functions",
      },
      {
        icon: Language,
        quantity: "Millions",
        text: "of function runs/month",
      },
      {
        icon: UsersGroup,
        quantity: "Custom",
        text: "seats",
      },
      {
        icon: ListCheck,
        quantity: "6+ month",
        text: "log retention",
      },
      {
        icon: ListCheck,
        quantity: "6+ month",
        text: "log retention",
      },
      {
        icon: UserVoice,
        quantity: "Dedicated",
        text: "support",
      },
    ],
  },
];

export default function Pricing() {
  const toggleFAQ = (e) => {
    e.currentTarget.classList.toggle("active");
  };

  return (
    <>
      <Head>
        <title>Inngest → programmable event platform pricing</title>
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
        {/* <p>
          Save weeks of development with our event-driven platform.
          <br />
          Get started in seconds for free, with plans that grow as you scale.
        </p> */}
      </Hero>

      <Content>
        <Block color="primary">
          <PlanHeader flexDirection="row">
            <div>
              <h3>{PLANS[0].name}</h3>
              <p>{PLANS[0].description}</p>
            </div>
            <div>
              <Button kind="outlineHighContrast" href={PLANS[0].cta.href}>
                {PLANS[0].cta.text}
              </Button>
            </div>
          </PlanHeader>
          <IconList items={PLANS[0].features} />
        </Block>

        <Grid>
          {PLANS.slice(1).map((plan, idx) => (
            <PlanBlock key={plan.name}>
              <PlanHeader flexDirection="column">
                <div>
                  <h3>{plan.name}</h3>
                  <p>{plan.description}</p>
                  <p className="cost">{plan.cost}</p>
                </div>
              </PlanHeader>
              <IconList direction="vertical" items={plan.features} />
              <Button kind="outlineHighContrast" href={plan.cta.href}>
                {plan.cta.text}
              </Button>
            </PlanBlock>
          ))}
        </Grid>

        <AllPlansInfo>
          All plans include uncapped events per month, scheduled functions,
          support via Email & Discord.
        </AllPlansInfo>

        <FAQGrid>
          <h2 style={{ margin: "2rem 0" }}>FAQs</h2>
          <div onClick={toggleFAQ}>
            <h3>What's a "function?"</h3>
            <p>
              A function is a single serverless function or step function.
              Functions can run automatically (each time an event is received),
              on a schedule, or manually via forms (in the case of internal
              tools, or one-offs).
            </p>
            <p>
              Functions allow you to chain logic together, coordinate between
              events, and create complex user flows and operational logic for
              your company.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What languages do you support?</h3>
            <p>
              Every language. Anything that you can put in a Docker container we
              can run from JavaScript to Go to Python to Perl to Bash.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>Do you charge for events?</h3>
            <p>
              We're currently accepting all events, uncapped. Our "soft" limit
              is 1M events/mo across free and paid plans, and from 50M events/mo
              on custom plans.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What's a "function run?"</h3>
            <p>
              A run is a single executed step in a function. If your function
              only has 1 step, it's only 1 run
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What are the resources limits?</h3>
            <p>
              Free accounts are limited to 128mb of ram and a maximum runtime of
              10 seconds per function. Paid accounts can utilize 1GB of ram and
              have a runtime limit of 60 seconds per function. Advanced accounts
              can use up to 16GB of ram and can run functions for up to 6 hours;
              if you need this functionality{" "}
              <a href="/contact">get in touch with us</a>.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What if I need to run more functions?</h3>
            <p>
              That's okay! We'll alert you when you're nearing your cap and will
              throttle your functions after hitting the limit. You can purchase
              more capacity at the same rate as your plan includes.
            </p>
            <p>
              We also offer overage forgiveness for those times that we all know
              happen. We dislike surprise costs as much as you.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>What is event coordination, and how can I use it?</h3>
            <p>
              Event coordination allows you to wait for specific events from
              within a function. For example, after creating a shopping cart you
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
            <h3>Is there a limit to how often I can run scheduled function?</h3>
            <p>
              In the community edition, functions are limited to 15 minute
              intervals and may be throttled for fair usage. Paid plans can run
              scheduled functions every minute, and are not subject to
              throttling. Custom plans have their own dedicated pools for
              managing functions.
            </p>
          </div>

          <div onClick={toggleFAQ}>
            <h3>Can I self-host Inngest?</h3>
            <p>
              If you'd like to self-host Inngest or the executor platform{" "}
              <a href="/contact">reach out to us with your needs</a>.
            </p>
          </div>
        </FAQGrid>

        <Callout ctaRef="pricing-callout-end" />
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
  p {
    font-family: var(--font);
  }
`;

const PlanHeader = styled.div<{
  flexDirection: "row" | "column";
}>`
  display: flex;
  flex-direction: ${(props) => props.flexDirection || "row"};
  justify-content: space-between;
  margin-bottom: 1.5rem;

  @media (max-width: 800px) {
    flex-direction: column;
  }

  p {
    margin: 1rem 0;
  }

  .cost {
    font-size: 1.4rem;
  }
`;

const Grid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-gap: 2rem 2rem;
  margin: 2rem 0;

  small {
    opacity: 0.5;
    font-size: 100%;
  }

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
  }
`;

const PlanBlock = styled(Block)`
  display: flex;
  flex-direction: column;

  .icon-list {
    flex-grow: 1;
    margin-bottom: 2rem;
  }
`;

const AllPlansInfo = styled.p`
  margin: 2rem auto 4rem;
  max-width: 44rem;
  line-height: 1.5rem;
  text-align: center;
`;

const FAQGrid = styled.div`
  display: grid;
  max-width: 44rem;
  margin: 2rem auto 3rem;

  div {
    cursor: pointer;

    h3 {
      margin: 1rem 0;
      font-family: var(--font);
      font-size: 1.2em;
    }
    p:not(:last-of-type) {
      margin-bottom: 1rem;
    }
    p:last-of-type {
      margin: 0 0 2rem;
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
