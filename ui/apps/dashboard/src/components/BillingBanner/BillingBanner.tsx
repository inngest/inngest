import { entitlementUsage } from '@/components/Billing/data';
import { BillingBannerView } from './BillingBannerView';

export async function BillingBanner() {
  const entUsage = await entitlementUsage();

  return <BillingBannerView entitlementUsage={entUsage} />;
}
