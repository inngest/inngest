import { useEffect } from "react";
import styled from "@emotion/styled";
import Head from "next/head";
import Footer from "../shared/Footer";
import Nav from "../shared/legacy/nav";
import Content from "../shared/legacy/content";
import Callout from "../shared/legacy/Callout";

import Block from "../shared/legacy/Block";
import IconList from "../shared/legacy/IconList";
// import Button from "../shared/Button";

import Workflow from "../shared/Icons/Workflow";
import Language from "../shared/Icons/Language";
import ArrowCaretCircleRight from "../shared/Icons/ArrowCaretCircleRight";
import Plus from "../shared/Icons/Plus";
import ListCheck from "../shared/Icons/ListCheck";
import UserVoice from "../shared/Icons/UserVoice";

import Functions from "../shared/Icons/Play";
import UsersGroup from "../shared/Icons/UsersGroup";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import { FAQRow } from "src/shared/Pricing/FAQ";
import PlanCard from "src/shared/Pricing/PlanCard";

type Plan = {
  name: string;
  cost: string;
  costTime?: string;
  description: React.ReactFragment | string;
  popular?: boolean;
  cta: {
    href: string;
    text: string;
  };
  features: {
    quantity?: string;
    text: string;
  }[];
  resources: {
    ram: string;
    maxRuntime: string;
  };
};

const PLANS: Plan[] = [
  {
    name: "Hobby",
    cost: "$0",
    costTime: "/month",
    description: <>Get to market fast</>,
    cta: {
      href: "/sign-up?ref=pricing-hobby",
      text: "Start building",
    },
    features: [
      {
        quantity: "50",
        text: "Functions",
      },
      {
        quantity: "Unlimited",
        text: "Events",
      },
      {
        quantity: "50K",
        text: "Function runs/month",
      },
      {
        quantity: "1",
        text: "Seat",
      },
      {
        quantity: "1 day",
        text: "History",
      },
      {
        text: "Discord support",
      },
    ],
    resources: {
      ram: "1GB",
      maxRuntime: "6 hours",
    },
  },
  {
    name: "Team",
    cost: "$20",
    costTime: "/month",
    description: <>More room to grow</>,
    popular: true,
    cta: {
      href: "/sign-up?ref=pricing-team",
      text: "Start building",
    },
    features: [
      {
        quantity: "100",
        text: "Functions",
      },
      {
        quantity: "Unlimited",
        text: "Events",
      },
      {
        quantity: "50K - 10M",
        text: "Function runs/month",
      },
      {
        quantity: "20",
        text: "Seats",
      },
      {
        quantity: "7 days",
        text: "History",
      },
      {
        text: "Dedicated support",
      },
    ],
    resources: {
      ram: "1GB",
      maxRuntime: "6 hours",
    },
  },
  {
    name: "Enterprise",
    cost: "Flexible",
    description: <>Powerful access for any scale</>,
    cta: {
      href: "/contact?ref=pricing-advanced",
      text: "Get in touch",
    },
    features: [
      {
        quantity: "Unlimited",
        text: "Functions",
      },
      {
        quantity: "Unlimited",
        text: "Events",
      },
      {
        quantity: "10M+",
        text: "Function runs/month",
      },
      {
        quantity: "20+",
        text: "Seats",
      },
      {
        quantity: "90 Days",
        text: "History",
      },
      {
        text: "Dedicated support",
      },
    ],
    resources: {
      ram: "up to 16GB",
      maxRuntime: "6+ hours",
    },
  },
];

export async function getStaticProps() {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Pricing",
        description: "Simple pricing. Powerful functionality.",
      },
    },
  };
}

export default function Pricing() {
  const toggleFAQ = (e) => {
    e.currentTarget.classList.toggle("active");
  };

  return (
    <div className="bg-slate-1000 font-sans">
      <Header />

      <Container>
        <h1 className="text-3xl lg:text-5xl text-white mt-20 mb-28 font-semibold tracking-tight">
          Simple pricing.
          <br />
          Powerful functionality.
        </h1>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-y-8 lg:gap-8 text-center mb-20">
          <div className="md:col-span-2 rounded-lg flex flex-col gap-y-8 md:gap-y-0 md:flex-row items-stretch">
            <PlanCard content={PLANS[0]} />
            <PlanCard content={PLANS[1]} />
          </div>
          <div className="flex items-stretch">
            <PlanCard content={PLANS[2]} type="dark" />
          </div>
        </div>

        {/* <table className="text-slate-200 w-full table-fixed">
          <thead>
            <tr className="border-b border-slate-900">
              <th className="px-6 py-4"></th>
              {PLANS.map((plan, i) => (
                <th className="text-left px-6 py-4" key={i}>
                  <h2 className="text-lg flex items-center">
                    {plan.name}{" "}
                    {plan.popular && (
                      <span className="bg-indigo-600 rounded-full font-semibold text-xs px-2 py-1 inline-block ml-3">
                        Most Popular
                      </span>
                    )}
                  </h2>
                </th>
              ))}
            </tr>
            <tr>
              <th></th>
              {PLANS.map((plan, i) => (
                <th className="text-left px-6 py-8" key={i}>
                  <span className="block text-4xl mb-2">
                    {plan.cost}
                    <span className="text-sm text-slate-400 ml-1 font-medium">
                      {plan.costTime}
                    </span>
                  </span>
                  <span className="block mb-8 text-sm font-medium mt-2 text-slate-200">
                    {plan.description}
                  </span>
                  <Button arrow href={plan.cta.href} full>
                    {plan.cta.text}
                  </Button>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            <tr>
              {PLANS.map((plan, i) => (
                <td></td>
              ))}
            </tr>
          </tbody>
        </table> */}

        <div className="xl:grid xl:grid-cols-4 mt-20 pt-12 border-t border-slate-900">
          <div>
            <h2 className="text-white mb-6 xl:mb-0 text-4xl font-semibold leading-tight tracking-tight mt-10">
              Frequently <br className="hidden xl:block" />
              asked <br className="hidden xl:block" />
              questions
            </h2>
          </div>
          <div className="col-span-3 text-slate-100 grid grid-cols-1 md:grid-cols-2 gap-4 gap-x-12">
            <FAQRow question={`What's a "function"?`}>
              <p>
                A function is a single serverless function or step function.
                Functions can run automatically (each time an event is
                received), on a schedule, or manually via forms (in the case of
                internal tools, or one-offs).
              </p>
              <p>
                Functions allow you to chain logic together, coordinate between
                events, and create complex user flows and operational logic for
                your company.
              </p>
            </FAQRow>

            <FAQRow question="What is event coordination, and how can I use it?">
              <p>
                Event coordination allows you to wait for specific events from
                within a function. For example, after creating a shopping cart
                you might want to wait for the "order created" event for up to
                24 hours. If this event is received, the person who created the
                shopping cart checked out. If the event isn't received within 24
                hours, you can run logic to handle churn.
              </p>
              <p>
                We allow you to coordinate between events and wait up to 6
                months for new events. That's... quite some time.
              </p>
            </FAQRow>

            <FAQRow question="What languages do you support?">
              <p>
                Every language. Anything that you can put in a Docker container
                we can run from JavaScript to Go to Python to Perl to Bash.
              </p>
            </FAQRow>

            <FAQRow question="Do you charge for events?">
              <p>
                We're currently accepting all events, uncapped. Our "soft" limit
                is 1M events/mo across free and paid plans, and from 50M
                events/mo on custom plans.
              </p>
            </FAQRow>

            <FAQRow question={`What's a "function run?"`}>
              <p>
                A run is a single executed step in a function. If your function
                only has 1 step, it's only 1 run.
              </p>
            </FAQRow>

            <FAQRow question="What if I need higher resource limits?">
              <p>
                If you need something more than what is listed above or
                something custom,{" "}
                <a
                  className="text-indigo-500 hover:text-white hover:underline transition-all"
                  href="/contact"
                >
                  get in touch with us
                </a>
                .
              </p>
            </FAQRow>

            <FAQRow question="What if I need to run more functions?">
              <p>
                That's okay! We'll alert you when you're nearing your cap and
                will throttle your functions after hitting the limit. You can
                purchase more capacity at the same rate as your plan includes.
              </p>
              <p>
                We also offer overage forgiveness for those times that we all
                know happen. We dislike surprise costs as much as you.
              </p>
            </FAQRow>

            <FAQRow question="Is there a limit to how often I can run scheduled function?">
              <p>
                In the community edition, functions are limited to 15 minute
                intervals and may be throttled for fair usage. Paid plans can
                run scheduled functions every minute, and are not subject to
                throttling. Custom plans have their own dedicated pools for
                managing functions.
              </p>
            </FAQRow>

            <FAQRow question="Can I self-host Inngest?">
              <p>
                If you'd like to self-host Inngest or the executor platform{" "}
                <a
                  className="text-indigo-500 hover:text-white hover:underline transition-all"
                  href="/contact"
                >
                  reach out to us with your needs
                </a>
                .
              </p>
            </FAQRow>
          </div>
        </div>
      </Container>

      <Footer />
    </div>
  );
}

const PlanHeader = styled.div<{
  flexDirection: "row" | "column";
}>`
  display: flex;
  flex-direction: ${(props) => props.flexDirection || "row"};
  justify-content: space-between;
  margin-bottom: 1.5rem;

  @media (max-width: 1000px) {
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
  @media (max-width: 1000px) {
    grid-template-columns: 1fr 1fr;
  }

  @media (max-width: 680px) {
    grid-template-columns: 1fr;
  }
`;

const PlanBlock = styled(Block)`
  display: flex;
  flex-direction: column;

  background: linear-gradient(
    135deg,
    hsl(330deg 82% 10%) 0%,
    hsl(239deg 82% 10%) 100%
  );

  .icon-list {
    flex-grow: 1;
    margin-bottom: 2rem;
  }
`;

const FreePlanBlock = styled(PlanBlock)<{ visibleOn: string }>`
  grid-column: 1 / span 3;

  display: ${({ visibleOn }) => (visibleOn === "desktop" ? "flex" : "none")};

  color: var(--black);
  background: linear-gradient(
    135deg,
    hsl(332deg 30% 95%) 0%,
    hsl(240deg 30% 95%) 100%
  );

  .icon-list svg {
    color: var(--color-white);
  }

  @media (min-width: 1000px) {
    .icon-list {
      margin-bottom: 0;
    }
  }

  @media (max-width: 1000px) {
    display: ${({ visibleOn }) => (visibleOn !== "desktop" ? "flex" : "none")};
    grid-column: 1;
  }
`;

const SectionHeader = styled.div`
  margin: 4rem 0 2rem;
  text-align: center;
`;

const SectionInfo = styled.p`
  margin: 1rem auto;
  max-width: 44rem;
  line-height: 1.5rem;
  text-align: center;
`;

const ComparisonTable = styled.div`
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  grid-gap: 1rem;
  max-width: 840px;
  padding: 1rem 1.4rem;
  margin: 1rem auto;
  border: 1px solid var(--stroke-color);
  border-radius: var(--border-radius);

  .table-header {
    font-family: var(--font);
    font-weight: bold;
  }
  .plan-column {
    display: grid;
    grid-row-gap: 1rem;
  }

  // invert the columns on small screens
  @media (max-width: 840px) {
    grid-template-columns: 1fr;
    grid-template-rows: repeat(3, 1fr);
    grid-gap: 0.5rem;
    padding: 0.8rem 1rem;
    font-size: 0.75rem;

    .plan-column {
      grid-template-columns: repeat(3, 1fr);
      grid-gap: 0.5rem;
    }
  }
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
