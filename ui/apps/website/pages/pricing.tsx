import { Button } from 'src/shared/Button';
import CodeWindow from 'src/shared/CodeWindow';
import Header from 'src/shared/Header';
import InformationCircle from 'src/shared/Icons/InformationCircle';
import ComparisonTable from 'src/shared/Pricing/ComparisionTable';
import { FAQRow } from 'src/shared/Pricing/FAQ';
import PlanCard from 'src/shared/Pricing/PlanCard';
import Container from 'src/shared/layout/Container';

import Footer from '../shared/Footer';

export type Plan = {
  name: string;
  cost: {
    startsAt?: boolean;
    // Use numbers for calculators
    basePrice: number | string;
    included: number | string;
    additionalPrice: number | string | null;
    additionalRate?: number;
    period?: string;
    max?: number;
  };
  description: React.ReactFragment | string;
  hideFromCards?: boolean;
  popular?: boolean;
  cta: {
    href: string;
    text: string;
    shortText?: string;
  };
  features: {
    quantity?: string;
    text: string;
  }[];
};

type Feature = {
  name: string;
  all?: boolean | string; // All plans offer this
  plans?: {
    [key: string]: string | boolean;
  };
  heading?: boolean;
  infoUrl?: string;
};

const PLAN_NAMES = {
  free: 'Free tier',
  team: 'Team',
  startup: 'Startup',
  enterprise: 'Enterprise',
};

const PLANS: Plan[] = [
  {
    hideFromCards: true,
    name: PLAN_NAMES.free,
    cost: {
      basePrice: 0,
      included: 50_000,
      period: 'month',
      additionalPrice: null,
    },
    description: 'Build your side project',
    cta: {
      href: `${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=pricing-free`,
      text: 'Create an account',
    },
    features: [
      {
        quantity: 'Unlimited',
        text: 'Events',
      },
      {
        quantity: 'Unlimited',
        text: 'Seats',
      },
      {
        quantity: '25',
        text: 'Concurrent functions',
      },
      {
        quantity: '3 days',
        text: 'History',
      },
      {
        text: 'Discord support',
      },
    ],
  },
  {
    name: PLAN_NAMES.team,
    cost: {
      startsAt: true,
      basePrice: 50,
      included: 100_000,
      additionalPrice: 1,
      additionalRate: 10_000,
      period: 'month',
      max: 3_000_000,
    },
    description: 'Bring your product to life',
    cta: {
      href: `${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=pricing-team`,
      text: 'Start building',
    },
    features: [
      {
        quantity: 'Unlimited',
        text: 'Events',
      },
      {
        quantity: 'Unlimited',
        text: 'Seats',
      },
      {
        quantity: '100',
        text: 'Concurrent functions',
      },
      {
        quantity: '7 days',
        text: 'History',
      },
      {
        quantity: 'Discord Community',
        text: 'Support',
      },
      { text: '-' },
      { text: '-' },
      { text: '-' },
      { text: '-' },
      { text: '-' },
    ],
  },
  {
    name: PLAN_NAMES.startup,
    cost: {
      startsAt: true,
      basePrice: 350,
      included: 5_000_000,
      additionalPrice: 5,
      additionalRate: 200_000,
      period: 'month',
      max: 30_000_000,
    },
    description: 'Scale with us',
    popular: true,
    cta: {
      href: `${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=pricing-startup`,
      text: 'Start building',
    },
    features: [
      {
        quantity: 'Unlimited',
        text: 'Events',
      },
      {
        quantity: 'Unlimited',
        text: 'Seats',
      },
      {
        quantity: '500',
        text: 'Concurrent functions',
      },
      {
        quantity: '14 days',
        text: 'History',
      },
      {
        quantity: 'Email, Discord',
        text: 'Support',
      },
      { text: '-' },
      { text: '-' },
      { text: '-' },
      { text: '-' },
      { text: '-' },
    ],
  },
  {
    name: PLAN_NAMES.enterprise,
    cost: {
      basePrice: 'Custom',
      included: 'Custom',
      additionalPrice: null,
    },
    description: 'Powerful access for any scale',
    cta: {
      href: '/contact?ref=pricing-enterprise',
      text: 'Speak with solutions engineering',
      shortText: 'Book a demo',
    },
    features: [
      {
        quantity: 'Unlimited',
        text: 'Events',
      },
      {
        quantity: 'Unlimited',
        text: 'Seats',
      },
      {
        quantity: 'Custom',
        text: 'Concurrent functions',
      },
      {
        quantity: '90 days',
        text: 'History',
      },
      {
        quantity: 'SLAs, Dedicated Slack channel, Email',
        text: 'Support',
      },
      {
        quantity: 'Single sign-on',
        text: 'Account security',
      },
      {
        text: 'Dedicated customer success',
      },
      {
        quantity: 'Datadog, Salesforce (+Add on)',
        text: 'Integrations',
      },
      {
        text: 'Data warehouse exports (+Add on)',
      },
      {
        quantity: 'SOC2 report & HIPAA/BAA',
        text: 'Compliance',
      },
    ],
  },
];

function getPlan(planName: string): Plan {
  return PLANS.find((p) => p.name === planName);
}

function getPlanFeatureQuantity(planName: string, feature: string): string {
  return getPlan(planName)?.features.find((f) => f.text === feature)?.quantity || '';
}

function getPlanStepsMonth(plan: Plan): string {
  if (typeof plan.cost.basePrice === 'string') {
    return plan.cost.basePrice;
  }
  const base = plan.cost.included.toLocaleString(undefined, {
    notation: 'compact',
    compactDisplay: 'short',
  });
  if (!plan.cost.additionalPrice) {
    return `${base}`;
  }
  return `${base} + $${
    plan.cost.additionalPrice
  } per additional ${plan.cost.additionalRate.toLocaleString(undefined, {
    notation: 'compact',
    compactDisplay: 'short',
  })}`;
}

const FEATURES: Feature[] = [
  {
    name: 'Steps/month',
    plans: {
      [PLAN_NAMES.free]: getPlanStepsMonth(getPlan(PLAN_NAMES.free)),
      [PLAN_NAMES.team]: getPlanStepsMonth(getPlan(PLAN_NAMES.team)),
      [PLAN_NAMES.startup]: getPlanStepsMonth(getPlan(PLAN_NAMES.startup)),
      [PLAN_NAMES.enterprise]: getPlanStepsMonth(getPlan(PLAN_NAMES.enterprise)),
    },
  },
  {
    name: 'Events',
    all: 'Unlimited',
  },
  {
    name: 'Seats',
    all: 'Unlimited',
  },
  {
    name: 'Concurrent functions',
    plans: {
      [PLAN_NAMES.free]: getPlanFeatureQuantity(PLAN_NAMES.free, 'Concurrent functions'),
      [PLAN_NAMES.team]: getPlanFeatureQuantity(PLAN_NAMES.team, 'Concurrent functions'),
      [PLAN_NAMES.startup]: getPlanFeatureQuantity(PLAN_NAMES.startup, 'Concurrent functions'),
      [PLAN_NAMES.enterprise]: getPlanFeatureQuantity(
        PLAN_NAMES.enterprise,
        'Concurrent functions'
      ),
    },
  },
  {
    name: 'History (log retention)',
    plans: {
      [PLAN_NAMES.free]: getPlanFeatureQuantity(PLAN_NAMES.free, 'History'),
      [PLAN_NAMES.team]: getPlanFeatureQuantity(PLAN_NAMES.team, 'History'),
      [PLAN_NAMES.startup]: getPlanFeatureQuantity(PLAN_NAMES.startup, 'History'),
      [PLAN_NAMES.enterprise]: getPlanFeatureQuantity(PLAN_NAMES.enterprise, 'History'),
    },
  },
  {
    name: 'Max sleep duration',
    plans: {
      [PLAN_NAMES.free]: '7 days',
      [PLAN_NAMES.team]: '60 days',
      [PLAN_NAMES.startup]: '6 months',
      [PLAN_NAMES.enterprise]: '1 year',
    },
    infoUrl: '/docs/guides/enqueueing-future-jobs?ref=pricing',
  },
  {
    name: 'Features',
    heading: true,
  },
  {
    name: 'Automatic retries',
    all: true,
    infoUrl: '/docs/functions/retries?ref=pricing',
  },
  {
    name: 'Step functions',
    all: true,
    infoUrl: '/docs/reference/functions/step-run?ref=pricing',
  },
  {
    name: 'Scheduled functions',
    all: true,
    infoUrl: '/docs/guides/scheduled-functions?ref=pricing',
  },
  {
    name: 'Concurrency controls',
    all: true,
    infoUrl: '/docs/functions/concurrency?ref=pricing',
  },
  {
    name: 'Custom failure handlers',
    all: true,
    infoUrl: '/docs/reference/functions/handling-failures?ref=pricing',
  },
  {
    name: 'Parallel steps',
    all: true,
    infoUrl: '/docs/guides/step-parallelism?ref=pricing',
  },
  {
    name: 'Fan-out',
    all: true,
    infoUrl: '/docs/guides/fan-out-jobs?ref=pricing',
  },
  {
    name: 'Local dev server',
    all: true,
    infoUrl: '/docs/local-development?ref=pricing',
  },
  {
    name: 'Branch environments',
    all: true,
    infoUrl: '/docs/platform/environments?ref=pricing#branch-environments',
  },
  {
    name: 'Vercel integration',
    all: true,
    infoUrl: '/docs/deploy/vercel?ref=pricing',
  },
  {
    name: 'Data warehouse exports',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: '+ Add on',
    },
  },
  {
    name: 'Integrations',
    heading: true,
  },
  {
    name: 'Datadog',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: true,
    },
  },
  {
    name: 'Salesforce',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: '+ Add on',
    },
  },
  {
    name: 'Support',
    heading: true,
  },
  {
    name: 'Discord support',
    plans: {
      [PLAN_NAMES.free]: true,
      [PLAN_NAMES.team]: true,
      [PLAN_NAMES.startup]: true,
      [PLAN_NAMES.enterprise]: true,
    },
  },
  {
    name: 'Email support',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: true,
      [PLAN_NAMES.enterprise]: true,
    },
  },
  {
    name: 'Dedicated Slack channel support',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: true,
    },
  },
  {
    name: 'SLAs',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: true,
    },
  },
  {
    name: 'Dedicated customer success',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: true,
    },
  },
  {
    name: 'Solutions engineering',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: '+ Add on',
    },
  },
  {
    name: 'Security & Privacy',
    heading: true,
  },
  {
    name: 'HIPAA BAA',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: 'Available',
    },
  },
  {
    name: 'SOC2 report',
    plans: {
      [PLAN_NAMES.team]: false,
      [PLAN_NAMES.startup]: false,
      [PLAN_NAMES.enterprise]: true,
    },
  },
];

const stepExamples = {
  singleStep: `
  export default inngest.createFunction(
    { id: "send-welcome-email" },
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
    { id: "new-signup-drip-campaign" },
    { event: "app/user.signup" },
    async ({ event, step }) => {
      await step.run("send-welcome-email", async () => {
        await emailAPI.send({
          template: "welcome",
          to: event.user.email,
        });
      });

      await step.sleep("wait-before-tips", "3 days");

      await step.run("send-new-user-tips-email", async () => {
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
      designVersion: '2',
      meta: {
        title: 'Pricing',
        description: 'Simple pricing. Powerful functionality.',
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
          backgroundImage: 'url(/assets/pricing/table-bg.png)',
          backgroundPosition: 'center -30px',
          backgroundRepeat: 'no-repeat',
          backgroundSize: '1800px 1200px',
        }}
      >
        <Container>
          <h1 className="mb-16 mt-20 text-3xl font-semibold tracking-tight text-white lg:text-5xl">
            Simple pricing.
            <br />
            Powerful functionality.
          </h1>
          <div className="my-12 flex items-center justify-center">
            <div className="w-4xl relative min-w-[80%] max-w-3xl sm:min-w-0">
              <div className="inset-0 -z-0 mx-5 rotate-2 scale-x-[110%] rounded-lg bg-blue-500 opacity-20 lg:absolute"></div>
              <div
                style={{
                  backgroundImage: 'url(/assets/footer/footer-grid.svg)',
                  backgroundSize: 'cover',
                  backgroundPosition: 'right -60px top -160px',
                  backgroundRepeat: 'no-repeat',
                }}
                className="relative flex h-full w-full flex-col justify-between rounded-xl bg-blue-500/90"
              >
                <div className="flex flex-col items-center gap-6 px-4 py-4 text-left sm:flex-row">
                  <div>
                    <h3 className="mb-2 text-xl font-medium tracking-tight text-white lg:text-2xl">
                      {PLAN_NAMES.free}
                    </h3>
                    <p className="flex items-center text-sm text-sky-100">
                      {getPlan(PLAN_NAMES.free).cost.included} steps{' '}
                      <a
                        href="#what-is-a-function-step"
                        className="mx-1 text-slate-200 transition-all hover:text-white"
                      >
                        <InformationCircle size="1.2em" />
                      </a>{' '}
                      &mdash; {getPlanFeatureQuantity(PLAN_NAMES.free, 'Concurrent functions')}{' '}
                      concurrent functions &mdash;{' '}
                      {getPlanFeatureQuantity(PLAN_NAMES.free, 'History')} history
                    </p>
                  </div>
                  <div className="flex flex-col items-center gap-2">
                    <Button
                      href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=free`}
                      variant="tertiary"
                      arrow="right"
                      className="whitespace-nowrap"
                    >
                      Create an account
                    </Button>
                    <p className="text-xs text-slate-200">
                      <em>No credit-card required</em>
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className="mb-8 grid grid-cols-1 gap-y-8 text-center md:grid-cols-3 lg:grid-cols-3 lg:gap-0">
            {PLANS.filter((p) => p.hideFromCards !== true).map((p) => (
              <PlanCard key={p.name} content={p} variant={p.popular ? 'focus' : 'light'} />
            ))}
          </div>

          <div className="">
            {/* Step Comparison */}
            <h2
              id="what-is-a-function-step" // Used in PlanCard
              className="mb-4 mt-20 scroll-mt-32 text-4xl font-semibold tracking-tight text-white"
            >
              What is a step?
            </h2>

            <p className="mb-16 mt-8 max-w-4xl text-lg font-medium">
              <a href="/docs/functions/multi-step" className="text-white underline">
                Steps
              </a>{' '}
              are building blocks for logic in functions. They are individually retried, and only
              run once on success. They allow you to easily write complex logic in a single
              function.
            </p>

            <div className="mt-12 flex w-full flex-col items-start gap-8 lg:flex-row">
              <div className="w-full lg:max-w-md">
                <h3 className="text-lg font-semibold">Single-step function</h3>
                <p className="my-4">
                  This function does one thing. When a{' '}
                  <code className="bg-slate-800 text-sm text-slate-200">app/user.signup</code> event
                  is triggered, the function sends a welcome email. This is billed as 1 step.
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
                  In this example, when a{' '}
                  <code className="bg-slate-800 text-sm text-slate-200">app/user.signup</code> event
                  is triggered the function sends a welcome email, waits 3 days, then sends another
                  email - without any extra queues, state, or functions. This is billed as 3 steps.
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
            <div className="mt-12 w-full">
              <h3 className="text-2xl font-semibold">How can I estimate steps?</h3>
              <p className="my-4 max-w-3xl">
                You can use the volume of messages that you currently process in your queues to
                approximate your step usage. Additionally, add the number of cron jobs that you run
                if you aim to use Inngest for scheduling. <br />
                <br />
                <a href="/contact?ref=estimate-steps">Get in touch</a> if you want help estimating
                advanced use cases.
              </p>
            </div>
          </div>

          <ComparisonTable plans={PLANS} features={FEATURES} />

          <div className="mt-12 border-t border-slate-900 pt-12 xl:grid xl:grid-cols-4">
            <div>
              <h2
                id="faq"
                className="mb-6 mt-10 scroll-mt-32 text-4xl font-semibold leading-tight tracking-tight text-white xl:mb-0"
              >
                Frequently <br className="hidden xl:block" />
                asked <br className="hidden xl:block" />
                questions
              </h2>
            </div>
            <div className="col-span-3 grid grid-cols-1 gap-4 gap-x-12 text-slate-100 md:grid-cols-2">
              <FAQRow question={`What's a "function"?`}>
                <p>
                  A function is defined with the{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
                    href="/docs/functions"
                  >
                    Inngest SDK
                  </a>{' '}
                  using <code className="bg-slate-800 text-slate-200">createFunction</code> or
                  similar. A function can be triggered by an event or run on a schedule (cron).
                </p>
                <p>
                  Functions can contain multiple “steps” to reliably run parts of your function or
                  add functionality like sleeping/pausing a function for a period of time. You can
                  define a step using available tools in our SDKs like{' '}
                  <code className="bg-slate-800 text-slate-200">step.run</code>,{' '}
                  <code className="bg-slate-800 text-slate-200">step.sleep</code>,
                  <code className="bg-slate-800 text-slate-200">step.sleepUntil</code> and{' '}
                  <code className="bg-slate-800 text-slate-200">step.waitForEvent</code>. Read more
                  in our{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
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
                  Your functions are hosted in your existing application on{' '}
                  <span className="italic">any platform</span>. We’ll call your functions securely
                  via HTTP request on-demand.
                </p>
                <p>
                  Each function step is called as a separate HTTP request enabling things like
                  having a function <code className="bg-slate-800 text-slate-200">sleep</code> for
                  minutes, hours or days.
                </p>
              </FAQRow>
              <FAQRow question={`What are concurrency limits?`}>
                <p>
                  As Inngest runs your function any time an event is received, you may have any
                  number of events received within a short period of time (e.g. 10ms). Inngest can
                  run all of these functions concurrently (in parallel). Our free tier allows for up
                  to {getPlanFeatureQuantity('Free', 'Concurrent Functions')} concurrent functions
                  at a time. Our paid plans offer substantial concurrency to enable you to
                  parallelize workloads and keep your system efficient and performant.
                </p>
                <p>
                  Sleeps and other pauses do not count towards your concurrency limit as your
                  function isn't running while waiting.
                </p>
                <p>
                  See more details at{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
                    href="/docs/usage-limits/inngest"
                  >
                    usage limits
                  </a>{' '}
                  page.
                </p>
              </FAQRow>
              <FAQRow question={`Can I get a demo of the product?`}>
                <p>
                  Yes! We would be happy to demo Inngest for you and understand the needs of your
                  team.{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
                    href="/contact?ref=pricing-faq-demo"
                  >
                    Contact us here
                  </a>{' '}
                  to set up a call.
                </p>
              </FAQRow>
              <FAQRow question={`What languages do you support?`}>
                <p>
                  We currently have an SDK for JavaScript/TypeScript, but plan to expand to Go,
                  Python and others in the future.{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
                    href="https://roadmap.inngest.com/roadmap"
                  >
                    Share your feedback or up vote an specific language SDK here
                  </a>
                  .
                </p>
              </FAQRow>
              <FAQRow question={`How long can my functions run for?`}>
                <p>
                  Inngest functions are invoked via https, so each function step can run as long as
                  your platform or server supports, for example, Vercel's Pro plan runs functions
                  for up to 60 seconds which means that if your function needs to run longer than
                  that, you can break it up into multiple steps (see: What is a function step?).
                </p>
                <p>
                  See more details on our{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
                    href="/docs/usage-limits/inngest"
                  >
                    usage limits
                  </a>{' '}
                  documentation.
                </p>
              </FAQRow>
              <FAQRow question={`Can multiple functions be triggered by the same event?`}>
                <p>
                  Yep! Any number of functions can be triggered by the same event enabling useful{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
                    href="/patterns"
                  >
                    design patterns
                  </a>{' '}
                  like fan-out.
                </p>
              </FAQRow>
              <FAQRow question={`Do you charge for events?`}>
                <p>
                  Nope. You can send any event to Inngest via and SDK or a webhook at any scale. We
                  only charge for the code that you run: the “function steps.” We encourage teams to
                  send any/all events to the Inngest platform which then can allow them to add new
                  functions at any time.
                </p>
              </FAQRow>
              <FAQRow question={`Can I select a region for my data?`}>
                <p>
                  Not yet, but it's in our roadmap. If you have a specific roadmap in mind or would
                  like to be one of the first people to have access,{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
                    href="/contact?ref=pricing-faq-regions"
                  >
                    shoot us a message
                  </a>
                  .
                </p>
              </FAQRow>
              <FAQRow question={`Can I self host inngest?`}>
                <p>
                  Not yet, but we plan to offer this in the future. If you're interested in
                  self-hosting Inngest,{' '}
                  <a
                    className="text-indigo-400 transition-all hover:text-white hover:underline hover:decoration-white"
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
