import { ContextualBanner, type Severity } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';
import { formatDayString } from '@inngest/components/utils/date';

import { pathCreator } from '@/utils/urls';
import { type AccountPaymentStatus } from './types';
import { usePaymentStatus } from './usePaymentStatus';

function title(status: AccountPaymentStatus): string {
  switch (status.stage) {
    case 'PAYMENT_FAILED':
      return 'Your last payment failed';
    case 'PAST_DUE':
      return 'You have an overdue invoice';
    case 'FINAL_NOTICE':
      return `Your invoice is ${status.daysPastDue} days overdue`;
    case 'DOWNGRADE_PENDING':
      return status.actionDate
        ? `Your account will be downgraded on ${formatDayString(
            new Date(status.actionDate),
          )}`
        : 'Your account is scheduled to be downgraded';
    case 'DOWNGRADED':
      return 'Your account has been downgraded for non-payment';
    case 'SUSPENDED':
      return 'Your account is suspended for non-payment';
  }
}

// Detailed variant for the /billing Overview. Reuses the same query as the
// global banner (deduped by React Query), so it shows the per-invoice breakdown
// and support routing without a second request.
export function BillingPaymentStatusBanner() {
  const status = usePaymentStatus();
  if (!status) return null;

  const severity: Severity =
    status.severity === 'CRITICAL' ? 'error' : 'warning';
  const showSupport =
    status.stage === 'DOWNGRADED' || status.stage === 'SUSPENDED';

  return (
    <ContextualBanner
      severity={severity}
      className="mb-4"
      title={<strong>{title(status)}</strong>}
      cta={
        <Button
          appearance="outlined"
          size="small"
          kind={severity === 'error' ? 'danger' : 'secondary'}
          href={pathCreator.billing({
            tab: 'payments',
            ref: 'app-billing-overview-overdue',
          })}
          label="View invoices"
          className="shrink-0"
        />
      }
    >
      <div className="px-4 py-2">
        <p>
          Update your credit card to pay {status.amountDueLabel} in overdue
          invoices and keep your account open.
        </p>
        {status.overdueInvoices.length > 0 && (
          <ContextualBanner.List>
            {status.overdueInvoices.map((invoice) => (
              <li key={invoice.id}>
                {invoice.amountLabel} — due{' '}
                {formatDayString(new Date(invoice.dueAt))} (
                {invoice.daysPastDue} days overdue)
                {invoice.invoiceURL && (
                  <>
                    {' · '}
                    <ContextualBanner.Link
                      severity={severity}
                      href={invoice.invoiceURL}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      Pay invoice
                    </ContextualBanner.Link>
                  </>
                )}
              </li>
            ))}
          </ContextualBanner.List>
        )}
        {showSupport && (
          <p>
            Need help?{' '}
            <ContextualBanner.Link
              severity={severity}
              href={pathCreator.support({ ref: 'app-billing-overdue' })}
            >
              Contact support
            </ContextualBanner.Link>{' '}
            to restore your account.
          </p>
        )}
      </div>
    </ContextualBanner>
  );
}
