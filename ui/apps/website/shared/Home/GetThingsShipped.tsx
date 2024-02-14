import { useState } from 'react';
import classNames from 'src/utils/classNames';
import { stripIndent } from 'src/utils/string';

import CodeWindow from '../CodeWindow';
import { IconBackgroundTasks, IconJourney, IconScheduled, IconTools } from '../Icons/duotone';
import Container from '../layout/Container';
import Heading from './Heading';

export default function GetThingsShipped() {
  const tabs = [
    {
      title: 'Background Jobs',
      icon: IconBackgroundTasks,
      content: [
        {
          title: 'Out of the critical path',
          description:
            'Ensure your API is fast by running your code, asynchronously, in the background.',
        },
        {
          title: 'No queues or workers required',
          description:
            'Serverless background jobs mean you don’t need to set up queues or long-running workers.',
        },
      ],
      code: {
        title: 'sendConfirmationSMS.ts',
        content: stripIndent(`
          import { sendSMS } from "../twilioUtils";
          import { inngest } from "./client";

          export default inngest.createFunction(
            { id: "send-confirmation-sms" },
            { event: "app/request.confirmed" },
            async ({ event }) => {
              const result = await sendSMS({
                to: event.user.phone,
                message: "Your request has been confirmed!",
              });

              return {
                status: result.ok ? 200 : 500,
                body: \`SMS Sent (Message SID: \${result.sid})\`,
              };
            }
          );`),
      },
    },
    {
      title: 'Scheduled Jobs',
      icon: IconScheduled,
      content: [
        {
          title: 'Serverless cron jobs',
          description:
            'Run your function on a schedule to repeat hourly, daily, weekly or whatever you need.',
        },
        {
          title: 'No workarounds needed',
          description: "Tell Inngest when to run it and we'll take care of the rest.",
        },
      ],
      code: {
        title: 'sendWeeklyDigest.ts',
        content: stripIndent(`
          import { sendWeeklyDigestEmails } from "../emails";
          import { inngest } from "./client";

          export default inngest.createFunction(
            { id: "send-weekly-digest" },
            { cron: "0 9 * * MON" },
            sendWeeklyDigestEmails
          );`),
      },
    },
    {
      title: 'Webhooks',
      icon: '',
      content: [
        {
          title: 'Build reliable webhooks',
          description:
            'Inngest acts as a layer which can handle webhook events and that run your functions automatically.',
        },
        {
          title: 'Full observability',
          description:
            'The Inngest Cloud dashboard gives your complete observability into what event payloads were received and how your functions ran.',
        },
      ],
      code: {
        title: 'handleFailedPayments.ts',
        content: stripIndent(`
          import { downgradeAccount, findAccountByCustomerId } from "../accounts";
          import { sendFailedPaymentEmail } from "../emails";
          import { inngest } from "./client";

          export default inngest.createFunction(
            { id: "handle-failed-payments" },
            { name: "stripe/charge.failed" },
            async ({ event, step }) => {
              const account = await step.run("get-account", () =>
                findAccountByCustomerId(event.user.stripe_customer_id)
              );

              await Promise.all([
                sendFailedPaymentEmail(account.email),
                downgradeAccount(account.id),
              ]);
            }
          );`),
      },
    },
    {
      title: 'Internal Tools',
      icon: IconTools,
      content: [
        {
          title: 'Trigger scripts on demand',
          description:
            'Easily run necessary scripts on-demand triggered from tools like Retool or your own internal admin.',
        },
        {
          title: 'Run code with events from anywhere',
          description:
            'Slack or Stripe webhook events can trigger your code to run based off things like refunds or Slackbot interactions.',
        },
      ],
      code: {
        title: 'runUserDataBackfill.ts',
        content: stripIndent(`
          import { runBackfillForUser } from "../scripts";
          import { inngest } from "./client";

          export default inngest.createFunction(
            { id: "run-user-data-backfill" },
            { event: "retool/backfill.requested" },
            async ({ event }) => {
              const result = await runBackfillForUser(event.data.user_id);

              return {
                status: result.ok ? 200 : 500,
                body: \`Ran backfill for user \${event.data.user_id}\`,
              };
            }
          );`),
      },
    },
    {
      title: 'User Journey Automation',
      icon: IconJourney,
      content: [
        {
          title: 'User-behaviour driven',
          description:
            'Build out user-behavior driven flows for your product that are triggered by events sent from your app or third party integrations like drip email campaigns, re-activation campaigns, or reminders.',
        },
        {
          title: 'Step functions',
          description:
            'Add delays, connect multiple events, and build multi-step workflows to create amazing personalized experiences for your users.',
        },
      ],
      code: {
        title: 'userOnboardingCampaign.ts',
        content: stripIndent(`
          import { inngest } from "./client";

          export default inngest.createFunction(
            { id: "user-onboarding-campaign" },
            { event: "app/user.signup" },
            async ({ event, step }) => {
              await step.run("send-welcome-email", () =>
                sendEmail({
                  to: event.user.email,
                  template: "welcome",
                })
              );
              const profileComplete = await step.waitForEvent("wait-for-profile", {
                event: "app/user.profile.completed",
                timeout: "24h",
                match: "data.userId",
              });
              if (!profileComplete) {
                await step.run("send-reminder-email", () =>
                  sendEmail({
                    to: event.user.email,
                    template: "reminder",
                  })
                );
              }
            }
          );`),
      },
    },
    {
      title: 'Event-driven Systems',
      icon: '',
      content: [
        {
          title: 'Design around events',
          description:
            'Developers can send and subscribe to a variety of internal and external events, creating complex event-driven architectures without worrying about infrastructure and boilerplate.',
        },
        {
          title: 'Auto-generated event schemas',
          description:
            'Events are parsed and schemas are generated and versioned automatically as you send events giving more oversight to the events that power your application.',
        },
      ],
      code: {
        title: 'eventDriven.ts',
        content: stripIndent(`
          import { inngest } from "@/inngest";

          export const handleApptRequested = inngest.createFunction(
            { id: "..." },
            { event: "appointment.requested" },
            async () => { /* ... */ }
          );

          export const handleApptScheduled = inngest.createFunction(
            { id: "..." },
            { event: "appointment.scheduled" },
            async () => { /* ... */ }
          );

          export const handleApptConfirmed = inngest.createFunction(
            { id: "..." },
            { event: "appointment.confirmed" },
            async () => { /* ... */}
          );

          export const handleApptCancelled = inngest.createFunction(
            { id: "..." },
            { event: "appointment.cancelled" },
            async () => { /* ... */ }
          );`),
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
        <Heading
          title="Get things shipped"
          lede="We built all the features that you need to build powerful applications
        without having to re-invent the wheel."
          className="mx-auto max-w-3xl text-center"
        />
      </Container>

      <Container className="mb-96 mt-10 flex flex-col items-start lg:mt-20 xl:flex-row">
        <ul className="flex flex-wrap justify-start gap-1 pb-8 max-lg:self-center xl:w-[290px] xl:flex-col xl:gap-2 xl:pb-0 xl:pt-10">
          {tabs.map((tab, i) => (
            <li key={i}>
              <button
                className={classNames(
                  activeTab === i
                    ? `bg-indigo-500 text-slate-100`
                    : `bg-transparent text-slate-400 hover:bg-slate-900 hover:text-slate-200`,
                  `inline-flex rounded-full px-4 py-2 text-left text-sm font-medium transition-all`
                )}
                onClick={() => handleTabClick(i)}
              >
                {tab.title}
              </button>
            </li>
          ))}
        </ul>

        <div className="relative flex w-full rounded-lg pb-4 md:pb-0">
          <div className="absolute -left-10 -right-10 bottom-0 top-0 -z-0 mx-5 hidden rotate-1 rounded-lg opacity-20 md:block"></div>
          {tabs.map((tab, i) =>
            activeTab === i ? (
              <div
                className="z-10 flex flex-col overflow-hidden px-5 md:w-1/2 md:flex-row lg:pl-10 lg:pr-16"
                key={i}
              >
                <div className="flex flex-col gap-4 py-10 pr-8">
                  <h2 className="flex items-center gap-1 text-[1.375rem] font-semibold text-white">
                    {tab.title}
                  </h2>
                  {tab.content.map((content, j) => (
                    <div key={j} className="flex flex-col gap-0.5">
                      <h4 className="text-lg font-medium text-white">{content.title}</h4>
                      <p className="text-sm leading-6 text-indigo-200 ">{content.description}</p>
                    </div>
                  ))}
                </div>
                <div className="right-10 top-10 max-w-full overflow-x-scroll md:absolute md:w-1/2">
                  <CodeWindow
                    snippet={tab.code.content.trim()}
                    header={
                      <div className="flex px-5 py-2">
                        <div className="rounded-full bg-slate-950 px-4 py-1 text-sm font-medium text-white">
                          {tab.code.title}
                        </div>
                      </div>
                    }
                    className="w-full grow text-xs md:max-w-xl lg:text-[13px]"
                    style={{
                      background: `radial-gradient(114.31% 100% at 50% 0%, #131E38 0%, #0A1223 100%),
                        linear-gradient(180deg, rgba(255, 255, 255, 0.06) 0%, rgba(255, 255, 255, 0.02) 100%)`,
                    }}
                    showLineNumbers={true}
                  />
                </div>
              </div>
            ) : null
          )}
        </div>
      </Container>
    </>
  );
}
