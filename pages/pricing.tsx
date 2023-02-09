import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import ComparisonTable from "src/shared/Pricing/ComparisionTable";
import { FAQRow } from "src/shared/Pricing/FAQ";
import PlanCard from "src/shared/Pricing/PlanCard";
import Footer from "../shared/Footer";

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

type Feature = {
  name: string;
  plans: {
    [key: string]: string | boolean;
  };
};

const FEATURES: Feature[] = [
  {
    name: "Functions",
    plans: {
      Hobby: "50",
      Team: "100",
      Enterprise: "Unlimited",
    },
  },
  {
    name: "Events",
    plans: {
      Hobby: "Unlimited",
      Team: "Unlimited",
      Enterprise: "Unlimited",
    },
  },
  {
    name: "Function steps/month",
    plans: {
      Hobby: "50K",
      Team: "100K - 1M",
      Enterprise: "Unlimited",
    },
  },
  {
    name: "Seats",
    plans: {
      Hobby: "1",
      Team: "20",
      Enterprise: "20+",
    },
  },
  {
    name: "Concurrent Functions",
    plans: {
      Hobby: "1",
      Team: "100",
      Enterprise: "Custom",
    },
  },
  {
    name: "Automatic Retries",
    plans: {
      Hobby: true,
      Team: true,
      Enterprise: true,
    },
  },
  {
    name: "Step Functions",
    plans: {
      Hobby: true,
      Team: true,
      Enterprise: true,
    },
  },
  {
    name: "Scheduled Functions",
    plans: {
      Hobby: true,
      Team: true,
      Enterprise: true,
    },
  },
  {
    name: "Local Dev Server",
    plans: {
      Hobby: true,
      Team: true,
      Enterprise: true,
    },
  },
  {
    name: "Event Coordination",
    plans: {
      Hobby: true,
      Team: true,
      Enterprise: true,
    },
  },
  {
    name: "Versioning",
    plans: {
      Hobby: true,
      Team: true,
      Enterprise: true,
    },
  },
];

const PLANS: Plan[] = [
  {
    name: "Hobby",
    cost: "$0",
    costTime: "/month",
    description: "Bring your project to life",
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
        quantity: "50k",
        text: "Function steps/month",
      },
      {
        quantity: "1",
        text: "Concurrent Function",
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
    cost: "From $20*",
    costTime: "/month",
    description: "From Startup to scale-up",
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
        quantity: "100k - 10m",
        text: "Function steps/month",
      },
      {
        quantity: "100",
        text: "Concurrent Functions",
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
        text: "Email support",
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
    description: "Powerful access for any scale",
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
        quantity: "10m+",
        text: "Function steps/month",
      },
      {
        quantity: "Custom",
        text: "Concurrent Functions",
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
        text: "Email support",
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
  return (
    <div className="font-sans">
      <Header />
      <div
        style={{
          backgroundImage: "url(/assets/pricing/table-bg.png)",
          backgroundPosition: "center -30px",
          backgroundRepeat: "no-repeat",
          backgroundSize: "1800px 1200px",
        }}
      >
        <Container>
          <h1 className="text-3xl lg:text-5xl text-white mt-20 mb-28 font-semibold tracking-tight">
            Simple pricing.
            <br />
            Powerful functionality.
          </h1>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-y-8 lg:gap-8 text-center mb-8">
            <div className="md:col-span-2 rounded-lg flex flex-col gap-y-8 md:gap-y-0 md:flex-row items-stretch">
              <PlanCard content={PLANS[0]} />
              <PlanCard content={PLANS[1]} />
            </div>
            <div className="flex items-stretch">
              <PlanCard content={PLANS[2]} variant="dark" />
            </div>
          </div>

          <p className="text-slate-200 text-sm text-center">
            *Team plan starts at $20/month for 100,000 function steps (
            <a href="#faq">?</a>).
            <br />
            Additional function steps are available to purchase for $10 per
            100,000.
          </p>

          <ComparisonTable plans={PLANS} features={FEATURES} />

          <div className="xl:grid xl:grid-cols-4 mt-20 pt-12 border-t border-slate-900">
            <div>
              <h2
                id="faq"
                className="scroll-mt-32 text-white mb-6 xl:mb-0 text-4xl font-semibold leading-tight tracking-tight mt-10"
              >
                Frequently <br className="hidden xl:block" />
                asked <br className="hidden xl:block" />
                questions
              </h2>
            </div>
            <div className="col-span-3 text-slate-100 grid grid-cols-1 md:grid-cols-2 gap-4 gap-x-12">
              <FAQRow question={`What's a "function"?`}>
                <p>
                  A function is defined with the{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="/docs/functions"
                  >
                    Inngest SDK
                  </a>
                  using{" "}
                  <code className="bg-slate-800 text-slate-200">
                    createFunction
                  </code>{" "}
                  or similar. A function can be triggered by an event or run on
                  a schedule (cron).
                </p>
                <p>
                  Functions can contain multiple “steps” to reliably run parts
                  of your function or add functionality like sleeping/pausing a
                  function for a period of time. You can define a step using
                  available tools in our SDKs like{" "}
                  <code className="bg-slate-800 text-slate-200">step.run</code>,{" "}
                  <code className="bg-slate-800 text-slate-200">
                    step.sleep
                  </code>
                  ,
                  <code className="bg-slate-800 text-slate-200">
                    step.sleepUntil
                  </code>{" "}
                  and{" "}
                  <code className="bg-slate-800 text-slate-200">
                    step.waitForEvent
                  </code>
                  . Read more in our{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="/docs/functions/multi-step"
                  >
                    documentation
                  </a>
                  .
                </p>
              </FAQRow>

              <FAQRow question={`What's a "function step"?`}>
                <p>
                  Inngest functions can be broken down into separate parts, or
                  “steps” which run independently. Steps are defined using our
                  SDK’s{" "}
                  <code className="bg-slate-800 text-slate-200">step</code>{" "}
                  object.
                </p>
                <p>
                  For example, any code within{" "}
                  <code className="bg-slate-800 text-slate-200">step.run</code>{" "}
                  will be retried up to 3 times independently of the rest of
                  your code ensuring your function is reliable. You can also add
                  delays in the middle of your functions for minutes, hours or
                  days using{" "}
                  <code className="bg-slate-800 text-slate-200">
                    step.sleep
                  </code>{" "}
                  or{" "}
                  <code className="bg-slate-800 text-slate-200">
                    step.sleepUntil
                  </code>
                  . You function can also wait for additional events to trigger
                  additional logic with{" "}
                  <code className="bg-slate-800 text-slate-200">
                    step.waitForEvent
                  </code>{" "}
                  which enables you to build functions that pause while they
                  wait for additional input. Read more about steps{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="/docs/functions/multi-step"
                  >
                    here
                  </a>
                  .
                </p>
              </FAQRow>

              <FAQRow question={`How are "function steps" billed?`}>
                <p>
                  Since Inngest invokes and individually retries each function
                  step, each time a step is called, it counts towards your
                  monthly limit. If a function is retried 3 times, that counts
                  for 3 function steps billed.
                </p>
                <p>
                  <strong className="text-slate-200">Scenario 1:</strong>
                  "Function A" does not use any{" "}
                  <code className="bg-slate-800 text-slate-200">step</code>{" "}
                  tools, it is considered a "single step function." If it is
                  called once and is completed successfully, that is 1 function
                  step.
                </p>
                <p>
                  <strong className="text-slate-200">Scenario 2:</strong>{" "}
                  "Function B" has 3 steps defined using both{" "}
                  <code className="bg-slate-800 text-slate-200">step.run</code>{" "}
                  and
                  <code className="bg-slate-800 text-slate-200">
                    step.sleep
                  </code>
                  . If it is called once and is completed successfully, that is
                  3 function steps.
                </p>
                <p>
                  <strong className="text-slate-200">Scenario 3:</strong>{" "}
                  "Function C" has 3 steps defined using both{" "}
                  <code className="bg-slate-800 text-slate-200">step.run</code>.
                  If it is called once and the first step succeeds, but the
                  second step fails 3 times due to{" "}
                  <a href="/docs/functions/retries">retries</a>, that is 4
                  function steps. The last step is never called due to the
                  failure, so it is not billed.
                </p>
              </FAQRow>

              <FAQRow question={`How are my functions run?`}>
                <p>
                  Your functions are hosted in your existing application on{" "}
                  <span className="italic">any platform</span>. We’ll call your
                  functions securely via HTTP request on-demand.
                </p>
                <p>
                  Each function step is called as a separate HTTP request
                  enabling things like having a function{" "}
                  <code className="bg-slate-800 text-slate-200">sleep</code> for
                  minutes, hours or days.
                </p>
              </FAQRow>
              <FAQRow question={`What are concurrency limits?`}>
                <p>
                  As Inngest runs your function any time an event is received,
                  you may have any number of events received within a short
                  period of time (e.g. 10ms). Inngest can run all of these
                  functions concurrently (in parallel). Our Hobby plan only
                  allows one function to run at a time. Our paid plans offer
                  substantial concurrency to enable you to parallelize workloads
                  and keep your system efficient and performant.
                </p>
                <p>
                  Sleeps and other pauses do not count towards your concurrency
                  limit as your function isn't running while waiting.
                </p>
              </FAQRow>
              <FAQRow question={`Can I get a demo of the product?`}>
                <p>
                  Yes! We would be happy to demo Inngest for you and understand
                  the needs of your team. Email us at{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="mailto:hello@inngest.com"
                  >
                    hello@inngest.com
                  </a>{" "}
                  to set up a call.
                </p>
              </FAQRow>
              <FAQRow question={`What languages do you support?`}>
                <p>
                  We currently have an SDK for JavaScript/TypeScript, but plan
                  to expand to Go, Python and others in the future.{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="mailto:hello@inngest.com"
                  >
                    Let us know
                  </a>{" "}
                  if you're interested in an SDK that we don't currently have.
                  up a call.
                </p>
              </FAQRow>
              <FAQRow question={`How long can my functions run for?`}>
                <p>
                  Inngest functions are invoked via http, so each function step
                  can run as long as your platform or server supports, for
                  example, Vercel’s Pro plan runs functions for up to 60 seconds
                  which means that if your function needs to run longer than
                  that, you can break it up into multiple steps (see: What is a
                  function step?).
                </p>
              </FAQRow>
              <FAQRow
                question={`Can multiple functions be triggered by the same event?`}
              >
                <p>
                  Yep! Any number of functions can be triggered by the same
                  event enabling useful{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="/patterns"
                  >
                    design patterns
                  </a>{" "}
                  like fan-out.
                </p>
              </FAQRow>
              <FAQRow question={`Do you charge for events?`}>
                <p>
                  Nope. You can send any event to Inngest via and SDK or a
                  webhook at any scale. We only charge for the code that you
                  run: the “function steps.” We encourage teams to send any/all
                  events to the Inngest platform which then can allow them to
                  add new functions at any time.
                </p>
              </FAQRow>
              <FAQRow question={`Can I select a region for my data?`}>
                <p>
                  Not yet, but it’s in our roadmap. If you have a specific
                  roadmap in mind or would like to be one of the first people to
                  have access,{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="mailto:hello@inngest.com"
                  >
                    shoot us a message
                  </a>
                  .
                </p>
              </FAQRow>
              <FAQRow question={`Can I self host inngest?`}>
                <p>
                  If you're interested in self-hosting Inngest,{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="mailto:hello@inngest.com"
                  >
                    reach out with your needs
                  </a>
                  .
                </p>
              </FAQRow>
            </div>
          </div>
        </Container>
      </div>

      <Footer />
    </div>
  );
}
