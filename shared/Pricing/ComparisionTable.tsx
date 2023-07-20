import { Button } from "../Button";
import InformationCircle from "src/shared/Icons/InformationCircle";

export default function ComparisonTable({ plans, features }) {
  const visiblePlans = plans.filter((p) => p.showInTable !== false);
  return (
    <div className="hidden lg:block">
      <h2 className="text-white mt-32 mb-8 text-4xl font-semibold">
        Compare all plans
      </h2>
      <table className="text-slate-200 w-full table-fixed">
        <thead>
          <tr className="border-b border-slate-900">
            <th className="px-6 py-4"></th>
            {visiblePlans.map((plan, i) => (
              <th className="text-left px-6 py-4" key={i}>
                <h2 className="text-lg flex items-center">
                  {plan.name}{" "}
                  {plan.popular && (
                    <span className="bg-indigo-500 rounded-full font-semibold text-xs px-2 py-1 inline-block ml-3">
                      Most popular
                    </span>
                  )}
                </h2>
              </th>
            ))}
          </tr>
          <tr>
            <th></th>
            {visiblePlans.map((plan, i) => (
              <th className="text-left px-6 py-8" key={i}>
                <span className="block text-xs text-slate-400 font-medium mb-1">
                  {plan.cost.startsAt ? `Starting at` : <>&nbsp;</>}
                </span>
                <span className="block text-4xl mb-2">
                  {plan.cost.basePrice}
                  {!!plan.cost.period && (
                    <span className="text-sm text-slate-400 ml-1 font-medium">
                      /{plan.cost.period}
                    </span>
                  )}
                </span>
                <span className="block mb-8 text-sm font-medium mt-2 text-slate-200">
                  {plan.description}
                </span>
                <Button arrow="right" href={plan.cta.href} full>
                  {plan.cta.shortText || plan.cta.text}
                </Button>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {features.map((feature, i) => (
            <tr key={i} className="h-14 border-t border-slate-900">
              <td className="h-14 flex items-center font-medium">
                {feature.name}
                {Boolean(feature.infoUrl) && (
                  <a
                    href={feature.infoUrl}
                    className="ml-2 transition-all text-slate-500 hover:text-white"
                  >
                    <InformationCircle size="1em" />
                  </a>
                )}
              </td>
              {visiblePlans.map((plan, j) => {
                const value =
                  typeof feature.plans?.[plan.name] === "string"
                    ? feature.plans?.[plan.name]
                    : typeof feature.all === "string"
                    ? feature.all
                    : null;
                const bool =
                  typeof feature.plans?.[plan.name] === "boolean"
                    ? feature.plans?.[plan.name]
                    : typeof feature.all === "boolean"
                    ? feature.all
                    : null;

                return value ? (
                  <td key={j} className="px-6 text-sm font-medium">
                    {value}
                  </td>
                ) : (
                  <td className="px-6" key={j}>
                    {bool ? (
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                        className="w-5 h-5 text-green-400"
                      >
                        <path
                          fillRule="evenodd"
                          d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z"
                          clipRule="evenodd"
                        />
                      </svg>
                    ) : (
                      <span className="text-slate-800">-</span>
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
