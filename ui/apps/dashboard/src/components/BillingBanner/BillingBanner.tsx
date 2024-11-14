import { entitlementUsage } from '@/components/Billing/data';
import { BillingBannerView } from './BillingBannerView';

export async function BillingBanner() {
  let entUsage;
  try {
    entUsage = (await entitlementUsage()).account.entitlementUsage;
  } catch (e) {
    console.error(e);
    return null;
  }

  return <BillingBannerView entitlementUsage={entUsage} />;
}
