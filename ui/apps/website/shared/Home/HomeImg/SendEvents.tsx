import { useState } from 'react';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { atomOneDark as syntaxThemeDark } from 'react-syntax-highlighter/dist/cjs/styles/hljs';
import classNames from 'src/utils/classNames';
import { stripIndent } from 'src/utils/string';

export default function SendEvents() {
  const [activeTab, setActiveTab] = useState(1);

  const tabs = [
    {
      title: 'Custom Event',
      payload: stripIndent(`inngest.send({
        name: "app/user.signup",
        data: { userId: "...", email: "..." },
      });`),
      fnName: 'Create Function',
      fnVersion: 27,
      fnID: '01GGG522ZATDGVQBCND4ZEAS6Z',
      code: stripIndent(`inngest.createFunction(
        { name: "post-signup" },
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
      title: 'Webhook',
      payload: stripIndent(`{
        name: "stripe/charge.failed",
        data: { ... }
      }`),
      fnName: 'Payment failed handler',
      fnVersion: 27,
      fnID: '01GGG522ZATDGVQBCND4ZEAS6Z',
      code: stripIndent(`import { downgradeAccount, findAccountByCustomerId } from "../accounts";
      import { sendFailedPaymentEmail } from "../emails";
      import { inngest } from "./client";

      export default inngest.createFunction(
        { id: "handle-failed-payments" },
        { event: "stripe/charge.failed" },
        async ({ event, step }) => {
          // step.run creates a reliable step which retries automatically,
          // storing the returned data in function state.
          const account = await step.run("get-account", () =>
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
          await step.sleep("wait-before-reminder", "3 days");

          await step.run("send-reminder", () => {
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
    <div className="bottom-10 right-20 z-10 -mt-10 flex flex-col gap-2 md:justify-end lg:absolute lg:items-end min-[1100px]:-mt-28 min-[1100px]:flex-row">
      <div className="w-full overflow-hidden rounded-lg border border-slate-700/30 bg-slate-800/50 shadow-lg backdrop-blur-md md:w-[400px] xl:mr-10 xl:w-[360px]">
        <div className="flex items-stretch justify-start gap-2 bg-slate-800/50 px-2">
          {tabs.map((tab, i) => (
            <button
              key={i}
              onClick={() => handleTabClick(i)}
              className={classNames(
                activeTab === i
                  ? `border-indigo-400 text-white`
                  : `border-transparent text-slate-400`,
                `border-b-[2px] px-2 py-2.5 text-center text-xs font-medium`
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
              codeTagProps={{ className: 'code-window' }}
              customStyle={{
                backgroundColor: 'transparent',
                fontSize: '0.7rem',
                padding: '1rem',
              }}
            >
              {tab.payload.trim()}
            </SyntaxHighlighter>
          ) : null
        )}
      </div>
      <div className="overflow-hidden rounded-lg border border-slate-700/30 bg-slate-800/50 shadow-lg backdrop-blur-md xl:w-[540px]">
        {tabs.map((tab, i) =>
          activeTab === i ? (
            <div key={i}>
              <div className="w-full bg-slate-800/50 px-4 py-3">
                <div className="flex items-start justify-between">
                  <div>
                    <span className="mb-1 block text-xs text-slate-400">30 seconds ago</span>
                    <h6 className="mb-0.5 text-sm text-slate-50">{tab.fnName}</h6>
                  </div>
                  <span className=" rounded bg-slate-900/50 px-2 py-1.5 text-xs font-bold leading-none text-slate-300">
                    {tab.fnVersion}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="hidden text-xs text-slate-400 md:block">{tab.fnID}</span>
                  <span className="flex items-center text-xs text-slate-200">
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
                codeTagProps={{ className: 'code-window' }}
                // className="hello"
                customStyle={{
                  backgroundColor: 'transparent',
                  fontSize: '0.7rem',
                  padding: '1.5rem',
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
