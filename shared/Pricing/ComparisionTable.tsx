import { Button } from "../Button";

export default function ComparisonTable({ plans, features }) {
  return (
    <div className="hidden md:block">
      <table className="text-slate-200 w-full table-fixed">
        <thead>
          <tr className="border-b border-slate-900">
            <th className="px-6 py-4"></th>
            {plans.map((plan, i) => (
              <th className="text-left px-6 py-4" key={i}>
                <h2 className="text-lg flex items-center">
                  {plan.name}{" "}
                  {plan.popular && (
                    <span className="bg-indigo-500 rounded-full font-semibold text-xs px-2 py-1 inline-block ml-3">
                      Most Popular
                    </span>
                  )}
                </h2>
              </th>
            ))}
          </tr>
          <tr>
            <th></th>
            {plans.map((plan, i) => (
              <th className="text-left px-6 py-8" key={i}>
                <span className="block text-4xl mb-2">
                  {plan.cost}
                  <span className="text-sm text-slate-400 ml-1 font-medium">
                    {plan.costTime}
                  </span>
                </span>
                <span className="block mb-8 text-sm font-medium mt-2 text-slate-200">
                  {plan.description}
                </span>
                <Button arrow="right" href={plan.cta.href} full>
                  {plan.cta.text}
                </Button>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {features.map((feature, i) => (
            <tr key={i} className="h-14 border-t border-slate-900">
              <td className="font-medium">{feature.name}</td>
              {plans.map((plan, i) =>
                typeof feature.plans[plan.name] === "string" ? (
                  <td key={i} className="px-6 text-sm font-medium">
                    {feature.plans[plan.name]}
                  </td>
                ) : (
                  <>
                    {feature.plans[plan.name] === true ? (
                      <td className="px-6" key={i}>
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
                      </td>
                    ) : (
                      <td className="px-6 text-slate-700" key={i}>
                        -
                      </td>
                    )}
                  </>
                )
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
