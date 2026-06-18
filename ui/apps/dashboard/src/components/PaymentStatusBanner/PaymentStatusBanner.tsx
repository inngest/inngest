import { PaymentStatusBannerView } from './PaymentStatusBannerView';
import { usePaymentStatus } from './usePaymentStatus';

// Account-wide overdue-invoice banner, rendered at the top of the authenticated
// layout alongside ActiveBanners. Renders nothing when the account is in good
// standing or the feature flag is off.
export function PaymentStatusBanner() {
  const status = usePaymentStatus();

  if (!status) return null;
  return <PaymentStatusBannerView status={status} />;
}
