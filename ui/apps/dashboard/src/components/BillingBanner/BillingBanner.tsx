import { entitlementUsage } from '@/components/Billing/data';
import { BillingBannerView } from './BillingBannerView';

export async function BillingBanner() {
  let data;
  try {
    data = await entitlementUsage();
  } catch (error) {
    // Avoid crashing the whole page if the fetch fails.
    return null;
  }
  if (data.isCustomPlan) {
    // Accounts on custom plans (a.k.a. enterprise) should never see the banner.
    return null;
  }

  return <BillingBannerView entitlementUsage={data.entitlements} />;
}
