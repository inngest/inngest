import { useState } from "react";
import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";
import classNames from "src/utils/classNames";
import { stripIndent } from "src/utils/string";

export default function SendEvents() {
  const [activeTab, setActiveTab] = useState(1);

  const tabs = [
    {
      title: "Custom Event",
      payload: stripIndent(`inngest.send({
        name: "app/user.signup",
        data: { userId: "...", email: "..." },
      });`),
      fnName: "Create Function",
      fnVersion: 27,
      fnID: "01GGG522ZATDGVQBCND4ZEAS6Z",
      code: stripIndent(`inngest.createFunction(
        { name: "Post-signup" },
        { event: "app/user.signup" },
        async ({ event }) => {
          await sendEmail({
            email: event.user.email,
            template: "welcome",
          });
        }
      );`),
    },
    {
      title: "Webhook",
      payload: stripIndent(`{
        name: "stripe/charge.failed",
        data: { ... }
      }`),
      fnName: "Payment failed handler",
      fnVersion: 27,
      fnID: "01GGG522ZATDGVQBCND4ZEAS6Z",
      code: stripIndent(`import { downgradeAccount, findAccountByCustomerId } from "../accounts";
      import { sendFailedPaymentEmail } from "../emails";
      import { inngest } from "./client";

      export default inngest.createFunction(
        { name: "Handle failed payments" },
        { event: "stripe/charge.failed" },
        async ({ event, step }) => {
          // step.run creates a reliable step which retries automatically,
          // storing the returned data in function state.
          const account = await step.run("Get account", () =>
            findAccountByCustomerId(event.user.stripe_customer_id)
          );

          // Use the account stored in function state from the previous step.
          // This calls two steps in parallel both retrying independently.
          await Promise.all([
            sendFailedPaymentEmail(account.email),
            downgradeAccount(account.id),
          ]);

          // The function will be woken up in 3 days with full state
          // injected, on any platform - even serverless functions.
          await step.sleep("3 days");

          await step.run("Send reminder", () => {
            sendReminder(account.email);
          });
        }
      );`),
    },
  ];

  const handleTabClick = (tab) => {
    setActiveTab(tab);
  };

  return (
    <div className="-mt-10 min-[1100px]:-mt-28 lg:absolute z-10 bottom-10 right-20 flex flex-col min-[1100px]:flex-row gap-2 lg:items-end md:justify-end">
      <div className="w-full md:w-[400px] xl:w-[360px] xl:mr-10 bg-slate-800/50 backdrop-blur-md border border-slate-700/30 rounded-lg overflow-hidden shadow-lg">
        <div className="flex bg-slate-800/50 items-stretch justify-start gap-2 px-2">
          {tabs.map((tab, i) => (
            <button
              key={i}
              onClick={() => handleTabClick(i)}
              className={classNames(
                activeTab === i
                  ? `border-indigo-400 text-white`
                  : ` border-transparent text-slate-400`,
                `font-medium text-center text-xs py-2.5 px-2 border-b-[2px]`
              )}
            >
              {tab.title}
            </button>
          ))}
        </div>
        {tabs.map((tab, i) =>
          activeTab === i ? (
            <SyntaxHighlighter
              key={i}
              language="javascript"
              showLineNumbers={false}
              style={syntaxThemeDark}
              codeTagProps={{ className: "code-window" }}
              customStyle={{
                backgroundColor: "transparent",
                fontSize: "0.7rem",
                padding: "1rem",
              }}
            >
              {tab.payload.trim()}
            </SyntaxHighlighter>
          ) : null
        )}
      </div>
      <div className="xl:w-[540px] bg-slate-800/50 backdrop-blur-md border border-slate-700/30 rounded-lg overflow-hidden shadow-lg">
        {tabs.map((tab, i) =>
          activeTab === i ? (
            <div key={i}>
              <div className="w-full py-3 px-4 bg-slate-800/50">
                <div className="flex justify-between items-start">
                  <div>
                    <span className="text-slate-400 text-xs mb-1 block">
                      30 seconds ago
                    </span>
                    <h6 className="text-sm text-slate-50 mb-0.5">
                      {tab.fnName}
                    </h6>
                  </div>
                  <span className=" bg-slate-900/50 rounded text-xs text-slate-300 font-bold px-2 py-1.5 leading-none">
                    {tab.fnVersion}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-xs text-slate-400 hidden md:block">
                    {tab.fnID}
                  </span>
                  <span className="text-xs text-slate-200 flex items-center">
                    <svg width="11" height="10" className="mr-1.5">
                      <path
                        d="M.294 5.057a.9167.9167 0 0 1 1.2964 0l1.9869 1.9876L9.415 1.2069a.9167.9167 0 0 1 1.2964 1.2964l-6.482 6.4821a.9168.9168 0 0 1-1.2547.0393l-.0879-.0784L.294 6.3535a.9167.9167 0 0 1 0-1.2964Z"
                        fill="#5EEAD4"
                        fillRule="evenodd"
                      />
                    </svg>
                    Completed
                  </span>
                </div>
              </div>
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
                {tab.code.trim()}
              </SyntaxHighlighter>
            </div>
          ) : null
        )}
      </div>
    </div>
  );
}
