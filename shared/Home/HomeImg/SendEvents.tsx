import SyntaxHighlighter from "react-syntax-highlighter";
import { atomOneDark as syntaxThemeDark } from "react-syntax-highlighter/dist/cjs/styles/hljs";
import { useState } from "react";
import classNames from "src/utils/classNames";

export default function SendEvents() {
  const [activeTab, setActiveTab] = useState(0);

  const tabs = [
    {
      title: "Custom Event",
      payload: `inngest.send({
    name: "app/user.signup",
    data: { userId: “...”, email: “...” }
})`,
      fnName: "Create Function",
      fnVersion: 27,
      fnID: "01GGG522ZATDGVQBCND4ZEAS6Z",
      code: `createFunction("post-signup", "app/user.signup",
  function ({ event, tools }) {
    // Send the user an email
    tools.run("Send an email", async () => {
      await sendEmail({
        email: event.user.email,
        template: "welcome",
      })
   })
})`,
    },
    {
      title: "Webhook",
      payload: `{
  name: "stripe/charge.failed",
  data: { ... }
}`,
      fnName: "Create Function",
      fnVersion: 27,
      fnID: "01GGG522ZATDGVQBCND4ZEAS6Z",
      code: `import { createFunction } from "inngest"
import {
  findAccountByCustomerId, downgradeAccount
} from "../accounts"
import { sendFailedPaymentEmail } from "../emails"

export default createStepFunction(
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
  ];

  const handleTabClick = (tab) => {
    setActiveTab(tab);
  };

  return (
    <div className="-mt-10 md:-mt-28 xl:absolute bottom-10 right-20 flex flex-col md:flex-row gap-2 md:items-end md:justify-end">
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
              {tab.payload}
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
                {tab.code}
              </SyntaxHighlighter>
            </div>
          ) : null
        )}
      </div>
    </div>
  );
}
