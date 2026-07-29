import { useMemo } from 'react';
import { Banner, type Severity } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';
import { useBooleanLocalStorage } from '@inngest/components/hooks/useBooleanLocalStorage';
import { formatDayString } from '@inngest/components/utils/date';

import {
  type AccountPaymentStatus,
  type PaymentCollectionStage,
} from './types';
import { pathCreator } from '@/utils/urls';

// Critical stages demand action and stay pinned, matching the existing
// convention where error-severity banners are non-dismissible.
const dismissibleStages: Record<PaymentCollectionStage, boolean> = {
  PAYMENT_FAILED: true,
  PAST_DUE: true,
  FINAL_NOTICE: false,
  DOWNGRADE_PENDING: false,
  DOWNGRADED: false,
  SUSPENDED: false,
};

function message(status: AccountPaymentStatus): string {
  const due = status.amountDueLabel;
  switch (status.stage) {
    case 'PAYMENT_FAILED':
      return 'Your last payment failed. Update your credit card to keep your account open.';
    case 'PAST_DUE':
      return `You have an overdue invoice of ${due}. Update your credit card to pay it and keep your account open.`;
    case 'FINAL_NOTICE':
      return `Your invoice is ${status.daysPastDue} days overdue. Update your credit card now to keep your account open.`;
    case 'DOWNGRADE_PENDING':
      return `Your account will be downgraded${
        status.actionDate
          ? ` on ${formatDayString(new Date(status.actionDate))}`
          : ''
      } due to unpaid invoices (${due}). Update your credit card to keep your plan.`;
    case 'DOWNGRADED':
      return `Your account has been downgraded due to unpaid invoices (${due}). Update your credit card to restore your plan.`;
    case 'SUSPENDED':
      return `Your account is suspended due to unpaid invoices (${due}). Update your credit card or contact support to restore access.`;
  }
}

export function PaymentStatusBannerView({
  status,
}: {
  status: AccountPaymentStatus;
}) {
  const severity: Severity =
    status.severity === 'CRITICAL' ? 'error' : 'warning';
  const dismissible = dismissibleStages[status.stage];

  // Key on stage so dismissing a soft warning doesn't suppress a later, more
  // severe notice once the account escalates.
  const isVisible = useBooleanLocalStorage(
    `PaymentStatusBanner:visible:${status.stage}`,
    true,
  );

  const onDismiss = useMemo(() => {
    if (!dismissible) return;
    return () => {
      isVisible.set(false);
    };
  }, [dismissible, isVisible]);

  // Wait for localStorage to hydrate before deciding visibility.
  if (!isVisible.isReady) return null;
  if (dismissible && !isVisible.value) return null;

  return (
    <Banner
      severity={severity}
      onDismiss={onDismiss}
      cta={
        <Button
          appearance="outlined"
          size="small"
          kind={severity === 'error' ? 'danger' : 'secondary'}
          href={pathCreator.billing({ ref: 'app-overdue-invoice-banner' })}
          label="Update credit card"
          className="ml-3 mr-2 inline-flex shrink-0 align-middle"
        />
      }
    >
      <span className="block text-left">{message(status)}</span>
    </Banner>
  );
}
