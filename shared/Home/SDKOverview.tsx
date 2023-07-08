import Link from "next/link";
import Image from "next/image";

import Container from "../layout/Container";
import CodeWindow from "../CodeWindow";
import Heading from "./Heading";

const codeSnippet = `
inngest.createFunction(
  { name: "Handle payments" },
  { event: "api/invoice.created" },
  async ({ event, step }) => {

    // Wait until the next billing date
    await step.sleepUntil(event.data.invoiceDate);

    // Steps automatically retry on error, and only run
    // once on success - automatically, with no work.
    const charge = await step.run("Charge", async () => {
      return await stripe.charges.create({
        amount: event.data.amount,
      });
    });

    await step.run("Update db", async () => {
      await db.payments.upsert(charge)
    });

    await step.run("Send receipt", async () => {
      await resend.emails.send({
        to: event.user.email,
        subject: "Your receipt for Inngest",
      })
    });
  }
)
`;

const highlights = [
  {
    title: "Any codebase, zero infrastructure",
    description: `Add our SDK to your existing project to start building in minutes. Inngest works with all of your favorite frameworks, without any infrastructure.`,
  },
  {
    title: "Declarative jobs & workflows",
    description: `Write your background jobs in just a few lines of code. Skip all boilerplate, and never define queues or state again.`,
  },
  {
    title: "Simple primitives, infinite possibilities",
    description: `Learn our SDK in minutes, not weeks, to build even the most complex workflows faster than ever before.`,
  },
];

export default function SDKOverview() {
  return (
    <Container className="my-36 max-w-[1200px] flex flex-col lg:flex-row justify-center items-start gap-24 tracking-tight">
      <div className="mx-4 sm:mx-auto max-w-lg">
        <Heading
          title="Ship in hours, not weeks"
          lede={
            <>
              Build everything from simple tasks to long-lived workflows using
              our SDK. With Inngest, there is zero infrastructure to set up -
              just write code.
              <br />
              <br />
              <span className="text-sm">
                <em>
                  * On average, teams ship their first Inngest function to
                  production in 4 hrs and 44 minutes.
                </em>
              </span>
            </>
          }
        />

        <div className="mt-8 flex flex-col gap-5 max-w-[468px]">
          {highlights.map(({ title, description }, idx) => (
            <div
              className="py-6 px-9 relative bg-slate-950/50 bg-cover"
              style={{ backgroundImage: `url(/assets/textures/wave.svg)` }}
            >
              <h3 className="text-lg sm:text-xl font-semibold text-indigo-50">
                {title}
              </h3>
              <p className="mt-2 text-sm text-indigo-200">{description}</p>
              <CheckCircleIcon className="absolute -left-3.5 top-0 bottom-0 my-auto h-7 w-7 text-indigo-500" />
            </div>
          ))}
        </div>
      </div>

      <CodeWindow
        snippet={codeSnippet}
        header={
          <div className="flex py-2 px-5">
            <div className="py-1 px-4 rounded-full text-sm font-medium text-white bg-slate-950">
              handlePayments.ts
            </div>
          </div>
        }
        className="grow w-full md:max-w-xl text-xs lg:text-[13px]"
        style={{
          background: `radial-gradient(114.31% 100% at 50% 0%, #131E38 0%, #0A1223 100%),
            linear-gradient(180deg, rgba(255, 255, 255, 0.06) 0%, rgba(255, 255, 255, 0.02) 100%)`,
        }}
        showLineNumbers={true}
      />
    </Container>
  );
}

// Custom w/ white checkmark
function CheckCircleIcon({ className }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="currentColor"
      className={className || "h-6 w-6"}
    >
      <circle cx="12" cy="12" r="8" fill="white" />
      <path
        fill-rule="evenodd"
        d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12zm13.36-1.814a.75.75 0 10-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 00-1.06 1.06l2.25 2.25a.75.75 0 001.14-.094l3.75-5.25z"
        clip-rule="evenodd"
      />
    </svg>
  );
}
