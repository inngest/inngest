import { entitlementUsage as getEntitlementUsage } from '@/components/Billing/data';
import { BillingBannerView } from './BillingBannerView';

export async function BillingBanner() {
  const entitlementUsage = await getEntitlementUsage();

  return <BillingBannerView entitlementUsage={entitlementUsage} />;
}
