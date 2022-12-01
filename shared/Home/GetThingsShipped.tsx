import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";
import { useState } from "react";
import Container from "./Container";
import classNames from "src/utils/classNames";
import SectionHeader from "./SectionHeader";

export default function GetThingsShipped() {
  const tabs = [
    {
      title: "Background Jobs",
      content: [
        {
          title: "Out of the critical path",
          description:
            "Ensure your API is fast by running your code, asynchronously, in the background.",
        },
        {
          title: "No queues or workers required",
          description:
            "Serverless background jobs mean you don’t need to set up queues or long-running workers.",
        },
      ],
      code: {
        title: "sendConfirmationSMS.js",
        content: `import { createFunction } from "inngest"
import { sendSMS } from "../twilioUtils"

export default createFunction(
  "Send confirmation SMS",
  "app/request.confirmed",
  async ({ event }) => {
    const result = await sendSMS({
      to: event.user.phone,
      message: "Your request has been confirmed!",
    })
    return {
      status: result.ok ? 200 : 500,
      body: \`SMS Sent (Message SID: \${result.sid})\`
    }
  }
)`,
      },
    },
    {
      title: "Scheduled Jobs",
      content: [
        {
          title: "Serverless cron jobs",
          description:
            "Run your function on a schedule to repeat hourly, daily, weekly or whatever you need.",
        },
        {
          title: "No workarounds needed",
          description:
            "Tell Inngest when to run it and we'll take care of the rest.",
        },
      ],
      code: {
        title: "sendWeeklyDigest.js",
        content: `import { createScheduledFunction } from "inngest"
import { sendWeeklyDigestEmails } from "../emails"

export default createScheduledFunction(
  "Send Weekly Digest",
  "0 9 * * MON",
  sendWeeklyDigestEmails
)`,
      },
    },
    {
      title: "Webhooks",
      content: [
        {
          title: "Build reliable webhooks",
          description:
            "Inngest acts as a layer which can handle webhook events and that run your functions automatically.",
        },
        {
          title: "Full observability",
          description:
            "The Inngest Cloud dashboard gives your complete observability into what event payloads were received and how your functions ran.",
        },
      ],
      code: {
        title: "handleFailedPayments.js",
        content: `import { createFunction } from "inngest"
import {
  findAccountByCustomerId, downgradeAccount
} from "../accounts"
import { sendFailedPaymentEmail } from "../emails"

export default createFunction(
  "Handle failed payments",
  "stripe/charge.failed",
  async ({ event }) => {
    const account = await = findAccountByCustomerId(event.user.stripe_customer_id)
    await sendFailedPaymentEmail(account.email)
    await downgradeAccount(account.id)
    return { message: "success" }
  }
)`,
      },
    },
    {
      title: "Internal Tools",
      content: [
        {
          title: "Trigger scripts on demand",
          description:
            "Easily run necessary scripts on-demand triggered from tools like Retool or your own internal admin.",
        },
        {
          title: "Run code with events from anywhere",
          description:
            "Slack or Stripe webhook events can trigger your code to run based off things like refunds or Slackbot interactions.",
        },
      ],
      code: {
        title: "runUserDataBackfill.js",
        content: `import { createFunction } from "inngest"
import { runBackfillForUser } from "../scripts"

export default createFunction(
  "Run user data backfill",
  "retool/backfill.requested",
  async ({ event }) => {
    const result = await runBackfillForUser(event.data.user_id)
    return {
      status: result.ok ? 200 : 500,
      body: \`Ran backfill for user \${event.data.user_id}\`
    }
  }
)`,
      },
    },
    {
      title: "User Journey Automation",
      content: [
        {
          title: "User-behaviour driven",
          description:
            "Build out user-behavior driven flows for your product that are triggered by events sent from your app or third party integrations like drip email campaigns, re-activation campaigns, or reminders.",
        },
        {
          title: "Step functions",
          description:
            "Add delays, connect multiple events, and build multi-step workflows to create amazing personalized experiences for your users.",
        },
      ],
      code: {
        title: "userOnboardingCampaign.js",
        content: `import { createStepFunction } from "inngest"

export default createStepFunction(
  "User onboarding campaign",
  "app/user.signup",
  /*
    Coming soon!
    Join the feedback group on Discord
  */
)`,
      },
    },
    {
      title: "Event-driven Systems",
      content: [
        {
          title: "Design around events",
          description:
            "Developers can send and subscribe to a variety of internal and external events, creating complex event-driven architectures without worrying about infrastructure and boilerplate.",
        },
        {
          title: "Auto-generated event schemas",
          description:
            "Events are parsed and schemas are generated and versioned automatically as you send events giving more oversight to the events that power your application.",
        },
      ],
      code: {
        title: "eventDriven.js",
        content: `import { createFunction } from "inngest"

export const handleApptRequested = createFunction("...",
  "appointment.requested", // ...
)
export const handleApptScheduled = createFunction("...",
  "appointment.scheduled", // ...
)
export const handleApptConfirmed = createFunction("...",
  "appointment.confirmed", // ...
)
export const handleApptCancelled = createFunction("...",
  "appointment.cancelled", // ...
)`,
      },
    },
  ];

  const [activeTab, setActiveTab] = useState(0);

  const handleTabClick = (index) => {
    setActiveTab(index);
  };

  return (
    <>
      <Container className="mt-40">
        <SectionHeader
          title="Get things shipped"
          lede="We built all the features that you need to build powerful applications
        without having to re-invent the wheel."
        />
      </Container>

      <Container className="flex flex-col xl:flex-row items-start mt-10 lg:mt-20 mb-80">
        <ul className="flex xl:flex-col flex-wrap justify-start gap-1 xl:gap-2 xl:w-[290px] pb-8 xl:pb-0 xl:pt-4">
          {tabs.map((tab, i) => (
            <li key={i}>
              <button
                className={classNames(
                  activeTab === i
                    ? `bg-indigo-500 text-slate-100`
                    : `bg-transparent text-slate-400 hover:bg-slate-900 hover:text-slate-200`,
                  `py-2 px-4 rounded-full inline-flex text-left text-sm transition-all font-medium`
                )}
                onClick={() => handleTabClick(i)}
              >
                {tab.title}
              </button>
            </li>
          ))}
        </ul>

        <div className="w-full rounded-lg bg-indigo-600 pb-4 md:pb-0 flex relative">
          <div className="hidden md:block absolute top-0 bottom-0 -left-10 -right-10 rounded-lg bg-indigo-500 opacity-20 rotate-1 -z-0 mx-5"></div>
          {tabs.map((tab, i) =>
            activeTab === i ? (
              <div
                className="flex flex-col md:flex-row px-5 lg:pl-10 lg:pr-16 md:w-1/2 overflow-hidden z-10"
                key={i}
              >
                <div className="py-10 pr-8 flex flex-col gap-4">
                  <h2 className="text-white text-xl font-semibold">
                    {tab.title}
                  </h2>
                  {tab.content.map((content, j) => (
                    <div key={j} className="flex flex-col gap-0.5">
                      <h4 className="text-lg text-white font-medium">
                        {content.title}
                      </h4>
                      <p className="text-indigo-50 text-sm leading-6 ">
                        {content.description}
                      </p>
                    </div>
                  ))}
                </div>
                <div className="max-w-full overflow-x-scroll md:w-1/2 md:absolute right-10 top-10 bg-slate-950/80 backdrop-blur-md border border-slate-800/60 rounded-lg overflow-hidden shadow-lg">
                  <h6 className="text-slate-300 w-full bg-slate-950/50 text-center text-xs py-1.5 border-b border-slate-800/50">
                    {tab.code.title}
                  </h6>
                  {/* <pre className="px-4 py-3 overflow-x-scroll max-w-full">
                    <code className="text-xs text-slate-300">
                      {tab.code.content}
                    </code>
                  </pre> */}

                  <SyntaxHighlighter
                    language="javascript"
                    showLineNumbers={false}
                    style={syntaxThemeDark}
                    codeTagProps={{ className: "code-window" }}
                    // className="hello"
                    customStyle={{
                      backgroundColor: "transparent",
                      fontSize: "0.7rem",
                      padding: "1.5rem",
                    }}
                  >
                    {tab.code.content}
                  </SyntaxHighlighter>
                </div>
              </div>
            ) : null
          )}
        </div>
      </Container>
    </>
  );
}
