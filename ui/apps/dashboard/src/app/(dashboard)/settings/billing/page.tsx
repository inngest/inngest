import { graphql } from '@/gql';
import type { BillingPlan } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { BillableStepUsage } from './BillableStepUsage/BillableStepUsage';
import { transformData } from './BillableStepUsage/transformData';
import BillingInformation from './BillingInformation';
import BillingPlanSelector from './BillingPlanSelector';
import CurrentSubscription from './CurrentSubscription';
import PaymentMethod from './PaymentMethod';
import Payments from './Payments';
import { isEnterprisePlan, transformPlan } from './utils';

// This will move to the API as a custom plan at some point, for now we can hard code
const ENTERPRISE_PLAN: BillingPlan = {
  id: 'n/a',
  name: 'Enterprise',
  amount: Infinity,
  billingPeriod: 'month',
  features: {
    actions: '10m+',
    concurrency: 'Custom',
    log_retention: 90,
    users: '20+',
    workflows: 'Custom',
  },
};

const GetBillingInfoDocument = graphql(`
  query GetBillingInfo($prevMonth: Int!, $thisMonth: Int!) {
    account {
      billingEmail
      name
      plan {
        id
        name
        amount
        billingPeriod
        features
      }
      subscription {
        nextInvoiceDate
      }

      paymentMethods {
        brand
        last4
        expMonth
        expYear
        createdAt
        default
      }
    }

    plans {
      id
      name
      amount
      billingPeriod
      features
    }

    # TODO: Improve monthly data querying. We're always querying for the current
    # month and the previous month, which is not ideal. But we did this to
    # quickly get the "view previous month" feature out.
    billableStepTimeSeriesPrevMonth: billableStepTimeSeries(timeOptions: { month: $prevMonth }) {
      data {
        time
        value
      }
    }

    billableStepTimeSeriesThisMonth: billableStepTimeSeries(timeOptions: { month: $thisMonth }) {
      data {
        time
        value
      }
    }
  }
`);

export default async function Billing() {
  // const [month, setMonth] = useState(new Date().getUTCMonth() + 1);

  const response = await graphqlAPI.request(GetBillingInfoDocument, {
    // Use UTC because the API uses UTC.
    prevMonth: new Date().getUTCMonth(),
    thisMonth: new Date().getUTCMonth() + 1,
  });

  const currentPlan = response.account.plan || undefined;
  const paymentMethod = response.account.paymentMethods?.[0] || null;
  const basePlans = [...response.plans, ENTERPRISE_PLAN];
  const subscription = response.account.subscription;

  const billableStepDataPrevMonth = response.billableStepTimeSeriesPrevMonth[0]?.data;
  const billableStepDataThisMonth = response.billableStepTimeSeriesThisMonth[0]?.data;

  let includedStepCountLimit: number | undefined;
  if (typeof response.account.plan?.features.actions === 'number') {
    includedStepCountLimit = response.account.plan?.features.actions;
  }

  const { totalStepCount } = billableStepDataThisMonth
    ? transformData(billableStepDataThisMonth, includedStepCountLimit)
    : { totalStepCount: 0 };

  // Always sort enterprise plans (including trials) last no matter the amount
  const plans =
    basePlans
      ?.map((plan) => (plan ? transformPlan({ plan, currentPlan, usage: totalStepCount }) : null))
      .sort((a, b) => (a?.isPremium ? 1 : (a?.amount || 0) - (b?.amount || 0))) || [];
  const isCurrentPlanEnterprise = currentPlan != undefined && isEnterprisePlan(currentPlan);
  const freePlan = plans.find((p) => p?.isFreeTier);

  return (
    <div className="overflow-y-scroll">
      <div className="mx-auto max-w-screen-xl">
        <header className="border-b border-slate-200 py-6">
          <h1 className="text-2xl font-semibold">Billing</h1>
        </header>

        <div className="mt-6 grid grid-cols-2 gap-2.5 lg:grid-cols-4">
          <CurrentSubscription
            subscription={subscription || undefined}
            currentPlan={currentPlan || undefined}
            isCurrentPlanEnterprise={isCurrentPlanEnterprise}
            freePlan={freePlan || undefined}
          />
          <div className="col-span-3 pl-6">
            {billableStepDataPrevMonth && billableStepDataThisMonth && (
              <BillableStepUsage
                data={{
                  prevMonth: billableStepDataPrevMonth,
                  thisMonth: billableStepDataThisMonth,
                }}
                includedStepCountLimit={includedStepCountLimit}
              />
            )}
          </div>
        </div>

        <BillingPlanSelector plans={plans} isCurrentPlanEnterprise={isCurrentPlanEnterprise} />

        <section>
          <h2 id="payments" className="py-6 text-2xl font-semibold">
            Payments
          </h2>
          <div className="mb-14 grid grid-cols-3 gap-2.5 border-t border-slate-200 pt-14">
            <div className="col-span-2">
              {/* @ts-ignore */}
              <Payments />
            </div>
            <div>
              <BillingInformation
                billingEmail={response?.account?.billingEmail}
                accountName={response?.account?.name}
              />
              <PaymentMethod paymentMethod={paymentMethod} />
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="mt-1.5 grid grid-cols-2 items-center gap-5 text-sm leading-8 text-slate-600">
      <div className="font-medium">{label}</div>
      <div className="font-bold">{value}</div>
    </div>
  );
}
