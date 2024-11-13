import { getEntitlementUsage } from '@/components/Billing/actions';
import { BillingBannerView } from './BillingBannerView';

export async function BillingBanner() {
  const entitlementUsage = await getEntitlementUsage();
  if (!entitlementUsage) return;

  return <BillingBannerView entitlementUsage={entitlementUsage} />;
}
