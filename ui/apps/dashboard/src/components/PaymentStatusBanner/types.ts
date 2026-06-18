// Hand-written until the API ships `account.paymentStatus`. Once the field
// exists in the introspected schema, switch the query in usePaymentStatus.ts to
// the codegen `graphql()` tag and replace these with the generated
// `PaymentStatusQuery` types.

export type PaymentStatusSeverity = 'WARNING' | 'CRITICAL';

export type PaymentCollectionStage =
  | 'PAYMENT_FAILED'
  | 'PAST_DUE'
  | 'FINAL_NOTICE'
  | 'DOWNGRADE_PENDING'
  | 'DOWNGRADED'
  | 'SUSPENDED';

export type PaymentPendingAction = 'DOWNGRADE' | 'SUSPEND';

export type OverdueInvoice = {
  id: string;
  amountLabel: string;
  dueAt: string;
  daysPastDue: number;
  status: string;
  invoiceURL: string | null;
  failureReason: string | null;
};

export type AccountPaymentStatus = {
  severity: PaymentStatusSeverity;
  stage: PaymentCollectionStage;
  amountDueLabel: string;
  daysPastDue: number;
  hasFailedPayment: boolean;
  actionDate: string | null;
  pendingAction: PaymentPendingAction | null;
  resolveURL: string;
  overdueInvoices: OverdueInvoice[];
};
