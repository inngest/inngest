import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import ComparisonTable from "src/shared/Pricing/ComparisionTable";
import { FAQRow } from "src/shared/Pricing/FAQ";
import PlanCard from "src/shared/Pricing/PlanCard";
import Footer from "../shared/Footer";
import { Button } from "src/shared/Button";
import CodeWindow from "src/shared/CodeWindow";
import InformationCircle from "src/shared/Icons/InformationCircle";

type Plan = {
  name: string;
  cost: {
    startsAt?: boolean;
    basePrice: string;
    included: string;
    additionalPrice: string;
    additionalRate?: string;
    period: string;
  };
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
};

type Feature = {
  name: string;
  all?: boolean; // All plans offer this
  plans?: {
    [key: string]: string | boolean;
  };
  infoUrl?: string;
};

const PLANS: Plan[] = [
  {
    name: "Team",
    cost: {
      basePrice: "$20",
      included: "100k",
      additionalPrice: "$1",
      additionalRate: "10k",
      period: "month",
    },
    description: "Bring your product to life",
    cta: {
      href: "/sign-up?ref=pricing-team",
      text: "Start building",
    },
    features: [
      {
        quantity: "Unlimited",
        text: "Events",
      },
      {
        quantity: "Unlimited",
        text: "Seats",
      },
      {
        quantity: "100",
        text: "Concurrent Functions",
      },
      {
        quantity: "7 days",
        text: "History",
      },
      {
        text: "Discord Support",
      },
    ],
  },
  {
    name: "Startup",
    cost: {
      basePrice: "$149",
      included: "5M",
      additionalPrice: "$10",
      additionalRate: "1M",
      period: "month",
    },
    description: "Scale with us",
    popular: true,
    cta: {
      href: "/sign-up?ref=pricing-startup",
      text: "Start building",
    },
    features: [
      {
        quantity: "Unlimited",
        text: "Events",
      },
      {
        quantity: "Unlimited",
        text: "Seats",
      },
      {
        quantity: "500",
        text: "Concurrent Functions",
      },
      {
        quantity: "14 days",
        text: "History",
      },
      {
        text: "Discord + Email Support",
      },
    ],
  },
  {
    name: "Enterprise",
    cost: {
      startsAt: true,
      basePrice: "$1250",
      included: "Custom",
      additionalPrice: "custom",
      period: "month",
    },
    description: "Powerful access for any scale",
    cta: {
      href: "/contact?ref=pricing-enterprise",
      text: "Get in touch",
    },
    features: [
      {
        quantity: "Unlimited",
        text: "Events",
      },
      {
        quantity: "Unlimited",
        text: "Seats",
      },
      {
        quantity: "Custom",
        text: "Concurrent Functions",
      },
      {
        quantity: "90 Days",
        text: "History",
      },
      {
        text: "Discord + Email Support + SLAs",
      },
      {
        quantity: "HIPAA BAA Available",
        text: "Compliance",
      },
    ],
  },
];

function getPlan(planName: string): Plan {
  return PLANS.find((p) => p.name === planName);
}

function getPlanFeatureQuantity(planName: string, feature: string): string {
  return (
    getPlan(planName)?.features.find((f) => f.text === feature)?.quantity || ""
  );
}

const FEATURES: Feature[] = [
  {
    name: "Function Steps/Month",
    plans: {
      Team: `${getPlan("Team").cost.included} + ${
        getPlan("Team").cost.additionalPrice
      } per additional ${getPlan("Team").cost.additionalRate}`,
      Startup: `${getPlan("Startup").cost.included} + ${
        getPlan("Startup").cost.additionalPrice
      } per additional ${getPlan("Startup").cost.additionalRate}`,
      Enterprise: getPlan("Enterprise").cost.included,
    },
  },
  {
    name: "Events",
    plans: {
      Team: getPlanFeatureQuantity("Team", "Events"),
      Startup: getPlanFeatureQuantity("Startup", "Events"),
      Enterprise: getPlanFeatureQuantity("Enterprise", "Events"),
    },
  },
  {
    name: "Seats",
    plans: {
      Team: "Unlimited",
      Startup: "Unlimited",
      Enterprise: "Unlimited",
    },
  },
  {
    name: "Concurrent Functions",
    plans: {
      Team: getPlanFeatureQuantity("Team", "Concurrent Functions"),
      Startup: getPlanFeatureQuantity("Startup", "Concurrent Functions"),
      Enterprise: getPlanFeatureQuantity("Enterprise", "Concurrent Functions"),
    },
  },
  {
    name: "History (Log Retention)",
    plans: {
      Team: getPlanFeatureQuantity("Team", "History"),
      Startup: getPlanFeatureQuantity("Startup", "History"),
      Enterprise: getPlanFeatureQuantity("Enterprise", "History"),
    },
  },
  {
    name: "Automatic Retries",
    all: true,
    infoUrl: "/docs/functions/retries?ref=pricing",
  },
  {
    name: "Step Functions",
    all: true,
    infoUrl: "/docs/reference/functions/step-run?ref=pricing",
  },
  {
    name: "Scheduled Functions",
    all: true,
    infoUrl: "/docs/guides/scheduled-functions?ref=pricing",
  },
  {
    name: "Max Sleep Duration",
    plans: {
      Team: "60 days",
      Startup: "6 months",
      Enterprise: "1 year",
    },
    infoUrl: "/docs/guides/enqueueing-future-jobs?ref=pricing",
  },
  {
    name: "Concurrency Controls",
    all: true,
    infoUrl: "/docs/functions/concurrency?ref=pricing",
  },
  {
    name: "Custom Failure Handlers",
    all: true,
    infoUrl: "/docs/reference/functions/handling-failures?ref=pricing",
  },
  {
    name: "Parallel Steps",
    all: true,
    infoUrl: "/docs/guides/step-parallelism?ref=pricing",
  },
  {
    name: "Fan-Out",
    all: true,
    infoUrl: "/docs/guides/fan-out-jobs?ref=pricing",
  },
  {
    name: "Local Dev Server",
    all: true,
    infoUrl: "/docs/local-development?ref=pricing",
  },
  {
    name: "Vercel Integration",
    all: true,
    infoUrl: "/docs/deploy/vercel?ref=pricing",
  },
  {
    name: "Discord Support",
    plans: {
      Team: true,
      Startup: true,
      Enterprise: true,
    },
  },
  {
    name: "Email Support",
    plans: {
      Team: false,
      Startup: true,
      Enterprise: true,
    },
  },
  {
    name: "Support SLA",
    plans: {
      Team: false,
      Startup: false,
      Enterprise: true,
    },
  },
  {
    name: "Onboarding Support",
    plans: {
      Team: false,
      Startup: false,
      Enterprise: true,
    },
  },
  {
    name: "HIPPA BAA Available",
    plans: {
      Team: false,
      Startup: false,
      Enterprise: true,
    },
  },
];

const stepExamples = {
  singleStep: `
  export default inngest.createFunction(
    { name: "Send Welcome Email" },
    { event: "app/user.signup" },
    async ({ event, step }) => {
      await emailAPI.send({
        template: "welcome",
        to: event.user.email,
      });
    }
  );
  `,
  multiStep: `
  export default inngest.createFunction(
    { name: "New Signup Drip Campaign" },
    { event: "app/user.signup" },
    async ({ event, step }) => {
      await step.run("Send welcome email", async () => {
        await emailAPI.send({
          template: "welcome",
          to: event.user.email,
        });
      });

      await step.sleep("3 days");

      await step.run("Send new user tips email", async () => {
        await emailAPI.send({
          template: "new-user-tips",
          to: event.user.email,
        });
      });
    }
  );
  `,
};

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
          <h1 className="text-3xl lg:text-5xl text-white mt-20 mb-16 font-semibold tracking-tight">
            Simple pricing.
            <br />
            Powerful functionality.
          </h1>
          <div className="flex items-center justify-center my-12">
            <div className="w-4xl min-w-[80%] sm:min-w-0 max-w-3xl relative">
              <div className="lg:absolute inset-0 rounded-lg bg-blue-500 opacity-20 rotate-2 -z-0 scale-x-[110%] mx-5"></div>
              <div
                style={{
                  backgroundImage: "url(/assets/footer/footer-grid.svg)",
                  backgroundSize: "cover",
                  backgroundPosition: "right -60px top -160px",
                  backgroundRepeat: "no-repeat",
                }}
                className="flex flex-col justify-between bg-blue-500/90 rounded-xl relative w-full h-full"
              >
                <div className="py-4 px-4 flex flex-col sm:flex-row gap-6 items-center text-left">
                  <div>
                    <h3 className="text-white text-xl lg:text-2xl font-medium tracking-tight mb-2">
                      Free Tier
                    </h3>
                    <p className="flex items-center text-sky-100 text-sm">
                      50k Function Steps{" "}
                      <a
                        href="#what-is-a-function-step"
                        className="mx-1 transition-all text-slate-200 hover:text-white"
                      >
                        <InformationCircle size="1.2em" />
                      </a>{" "}
                      &mdash; 25 Concurrent Functions &mdash; 3 Days History
                    </p>
                  </div>
                  <div className="flex flex-col gap-2 items-center">
                    <Button
                      href="/sign-up?ref=free"
                      variant="tertiary"
                      arrow="right"
                      className="whitespace-nowrap"
                    >
                      Create an account
                    </Button>
                    <p className="text-slate-200 text-xs">
                      <em>No credit-card required</em>
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-3 gap-y-8 lg:gap-0 text-center mb-8">
            {/* <div className="md:col-span-2 rounded-lg flex flex-col gap-y-8 md:gap-y-0 md:flex-row items-stretch"> */}
            <PlanCard content={PLANS[0]} variant="light" />
            <PlanCard content={PLANS[1]} variant="focus" />
            <PlanCard content={PLANS[2]} variant="light" />
            {/* </div> */}
          </div>

          <div>
            {/* Step Comparison */}
            <h2
              id="what-is-a-function-step" // Used in PlanCard
              className="scroll-mt-32 mt-20 mb-4 text-white text-4xl font-semibold tracking-tight"
            >
              What is a Function Step?
            </h2>

            <p className="my-8 text-lg font-medium">
              A Function Step is a callable unit of an Inngest function.
            </p>

            <div className="max-w-5xl w-full mt-12 flex flex-col lg:flex-row gap-8 items-start">
              <div className="w-full lg:max-w-md">
                <h3 className="text-lg font-semibold">
                  Simple, single-step function
                </h3>
                <p className="my-4">
                  This function does one thing. When a{" "}
                  <code className="bg-slate-800 text-slate-200 text-sm">
                    app/user.signup
                  </code>{" "}
                  event is triggered, the function sends a welcome email. This
                  is billed as 1 step.
                </p>
                <CodeWindow
                  className="mt-4 w-full"
                  snippet={stepExamples.singleStep}
                  lineHighlights={[[4, 9]]}
                />
              </div>
              <div className="max-w-[100%]">
                <h3 className="text-lg font-semibold">Multi-step function</h3>
                <p className="my-4">
                  This function combines functionality typically spread across
                  multiple jobs and crons. When a{" "}
                  <code className="bg-slate-800 text-slate-200 text-sm">
                    app/user.signup
                  </code>{" "}
                  event is triggered, the function sends a welcome email, it
                  waits 3 days, then sends another email. This is billed as 3
                  steps.
                </p>
                <div>
                  <CodeWindow
                    className="mt-4 w-full"
                    snippet={stepExamples.multiStep}
                    lineHighlights={[
                      [5, 10],
                      [12, 12],
                      [14, 19],
                    ]}
                  />
                </div>
              </div>
            </div>
          </div>

          <ComparisonTable plans={PLANS} features={FEATURES} />

          <div className="xl:grid xl:grid-cols-4 mt-12 pt-12 border-t border-slate-900">
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

              {/* <FAQRow question={`What's a "function step"?`}>
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
              </FAQRow> */}

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
                <p>
                  See more details at{" "}
                  <a href="/docs/usage-limits/inngest">Usage Limits</a> page.
                </p>
              </FAQRow>
              <FAQRow question={`Can I get a demo of the product?`}>
                <p>
                  Yes! We would be happy to demo Inngest for you and understand
                  the needs of your team.{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="/contact?ref=pricing-faq-demo"
                  >
                    Contact us here
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
                    href="/contact?ref=pricing-faq-languages"
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
                <p>
                  See more details at{" "}
                  <a href="/docs/usage-limits/inngest">Usage Limits</a> page.
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
                  Not yet, but it's in our roadmap. If you have a specific
                  roadmap in mind or would like to be one of the first people to
                  have access,{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="/contact?ref=pricing-faq-regions"
                  >
                    shoot us a message
                  </a>
                  .
                </p>
              </FAQRow>
              <FAQRow question={`Can I self host inngest?`}>
                <p>
                  Not yet, but we plan to offer this in the future. If you're
                  interested in self-hosting Inngest,{" "}
                  <a
                    className="text-indigo-400 hover:text-white hover:underline hover:decoration-white transition-all"
                    href="/contact?ref=pricing-faq-self-hosting"
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
