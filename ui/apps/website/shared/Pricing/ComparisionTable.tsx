import InformationCircle from 'src/shared/Icons/InformationCircle';

import { Button } from '../Button';

export default function ComparisonTable({ plans, features }) {
  const visiblePlans = plans.filter((p) => p.showInTable !== false);
  return (
    <div className="hidden lg:block">
      <h2 className="mb-8 mt-32 text-4xl font-semibold text-white">Compare all plans</h2>
      <table className="w-full table-fixed text-slate-200 ">
        <thead>
          {/* Sticky header height */}
          <tr className="bg-slate-1000 top-[84px] border-b border-slate-900 md:sticky">
            <th className="px-6 py-4"></th>
            {visiblePlans.map((plan, i) => (
              <th className="px-6 py-4 text-left" key={i}>
                <h2 className="flex items-center text-lg">
                  {plan.name}{' '}
                  {plan.popular && (
                    <span className="ml-3 inline-block rounded-full bg-indigo-500 px-2 py-1 text-xs font-semibold">
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
              <th className="px-6 py-8 text-left" key={i}>
                <span className="mb-1 block text-xs font-medium text-slate-400">
                  {plan.cost.startsAt ? `Starting at` : <>&nbsp;</>}
                </span>
                <span className="mb-2 block text-4xl">
                  {typeof plan.cost.basePrice === 'string'
                    ? plan.cost.basePrice
                    : `$${plan.cost.basePrice}`}
                  {!!plan.cost.period && (
                    <span className="ml-1 text-sm font-medium text-slate-400">
                      /{plan.cost.period}
                    </span>
                  )}
                </span>
                <span className="mb-8 mt-2 block text-sm font-medium text-slate-200">
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
              <td
                className={`flex h-14 items-center font-medium ${
                  feature.heading && 'mt-6 text-lg font-bold'
                }`}
              >
                {feature.name}
                {Boolean(feature.infoUrl) && (
                  <a
                    href={feature.infoUrl}
                    className="ml-2 text-slate-500 transition-all hover:text-white"
                  >
                    <InformationCircle size="1em" />
                  </a>
                )}
              </td>
              {visiblePlans.map((plan, j) => {
                const value = feature.heading
                  ? ''
                  : typeof feature.plans?.[plan.name] === 'string'
                  ? feature.plans?.[plan.name]
                  : typeof feature.all === 'string'
                  ? feature.all
                  : null;
                const bool =
                  typeof feature.plans?.[plan.name] === 'boolean'
                    ? feature.plans?.[plan.name]
                    : typeof feature.all === 'boolean'
                    ? feature.all
                    : null;

                return typeof value === 'string' ? (
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
                        className="h-5 w-5 text-green-400"
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
        <tfoot>
          <tr>
            <td></td>
            {visiblePlans.map((plan, i) => (
              <td className="px-6 py-8 text-left" key={i}>
                <Button arrow="right" href={plan.cta.href} full>
                  {plan.cta.shortText || plan.cta.text}
                </Button>
              </td>
            ))}
          </tr>
        </tfoot>
      </table>
    </div>
  );
}
