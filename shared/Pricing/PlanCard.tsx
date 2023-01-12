import classNames from "src/utils/classNames";
import { Button } from "../Button";

export default function PlanCard({ variant = "light", content }) {
  const theme = {
    light: {
      cardBG: "bg-slate-100 last:bg-white",
      price: "text-indigo-500",
      row: "odd:bg-slate-400/10",
      primary: "text-slate-800",
      secondary: "text-slate-600",
      description: "text-slate-600",
    },
    dark: {
      cardBG: "bg-slate-900/90",
      price: "text-indigo-400",
      row: "odd:bg-slate-400/10",
      primary: "text-white",
      secondary: "text-slate-400",
      description: "text-slate-200",
    },
  };

  return (
    <div
      className={`w-full rounded-lg md:rounded-l-none md:rounded-r-none md:first:rounded-l-lg md:last:rounded-r-lg flex flex-col justify-between text-center ${theme[variant].cardBG}`}
    >
      <div className="pt-8">
        {content.popular && (
          <div className="-mt-11 mb-3.5 block">
            <div className=" bg-indigo-500 inline-block shadow-lg rounded-full text-white text-sm font-semibold tracking-tight leading-none py-2 px-4">
              Most Popular
            </div>
          </div>
        )}
        <h2 className={`text-lg font-semibold ${theme[variant].primary}`}>
          {content.name}
        </h2>
        <p
          className={`text-3xl mt-2 -mr-4 font-bold tracking-tight text-indigo-500 ${theme[variant].price}`}
        >
          {content.cost}
          <span
            className={`text-sm font-medium ml-0.5 ${theme[variant].secondary}`}
          >
            /mo
          </span>
        </p>
        <p
          className={`text-sm mt-2 font-medium  ${theme[variant].description}`}
        >
          {content.description}
        </p>
        <ul className="flex flex-col mt-6">
          {content.features.map((feature, i) => (
            <li
              key={i}
              className={`flex flex-col py-2.5 ${theme[variant].row}`}
            >
              {feature.quantity && (
                <span className={`font-semibold ${theme[variant].primary}`}>
                  {feature.quantity}
                </span>
              )}
              <span
                className={classNames(
                  feature.quantity
                    ? `font-medium text-sm ${theme[variant].secondary}`
                    : `font-semibold my-2 ${theme[variant].primary}`,
                  `  tracking-tight`
                )}
              >
                {feature.text}
              </span>
            </li>
          ))}
        </ul>
      </div>
      <div className="px-12 py-2 mt-4 mb-4">
        <Button href={content.cta.href} arrow="right" full>
          {content.cta.text}
        </Button>
      </div>
    </div>
  );
}
