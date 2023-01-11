import classNames from "src/utils/classNames";
import { Button } from "../Button";

export default function PlanCard({ type = "light", content }) {
  return (
    <div
      className={classNames(
        type === "dark"
          ? `bg-slate-900 text-slate-100`
          : `bg-slate-200 last:bg-white`,
        `w-full first:rounded-l-lg last:rounded-r-lg flex flex-col justify-between text-center `
      )}
    >
      <div className="pt-8">
        {content.popular && (
          <div className="-mt-10 mb-2.5 block">
            <div className=" bg-indigo-500 inline-block rounded-full text-white text-sm font-semibold tracking-tight leading-none py-2 px-4">
              Most Popular
            </div>
          </div>
        )}
        <h2 className="text-lg font-semibold">{content.name}</h2>
        <p className="text-3xl mt-2 font-bold tracking-tight text-indigo-500">
          {content.cost}
        </p>
        <p
          className={classNames(
            type === "dark" ? `text-slate-200` : `text-slate-600`,
            `text-sm mt-2 font-medium`
          )}
        >
          {content.description}
        </p>
        <ul className="flex flex-col mt-6">
          {content.features.map((feature) => (
            <li
              className={classNames(
                type === "dark" ? `odd:bg-slate-600/10` : `odd:bg-slate-400/10`,
                `flex flex-col py-2.5 `
              )}
            >
              {feature.quantity && (
                <span
                  className={classNames(
                    type == "dark" ? `text-slate-200` : `text-slate-800`,
                    `font-semibold`
                  )}
                >
                  {feature.quantity}
                </span>
              )}
              <span
                className={classNames(
                  feature.quantity
                    ? `font-medium text-sm text-slate-500`
                    : `font-semibold mt-2 text-slate-800`,
                  `  tracking-tight`
                )}
              >
                {feature.text}
              </span>
            </li>
          ))}
        </ul>
      </div>
      <div className="px-12 pb-4 mt-4 mb-4">
        <Button href={content.cta.href} arrow="right" full>
          {content.cta.text}
        </Button>
      </div>
    </div>
  );
}
