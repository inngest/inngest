// Generated via inngest init

export const UserReport = {
  FRAUDULENT: "fraudulent",
  SAFE: "safe",
} as const;
export type UserReport = typeof UserReport[keyof typeof UserReport];

export interface StripeChargeSucceeded {
  name: "stripe/charge.succeeded";
  data: {
    id: string;
    type: "charge.succeeded";
    object: string;
    api_version: string;
    created: number;
    data: {
      object: {
        amount_captured: number;
        receipt_number: unknown;
        receipt_url: string;
        source_transfer: unknown;
        statement_descriptor_suffix: unknown;
        transfer_data: unknown;
        amount: number;
        dispute: unknown;
        disputed: boolean;
        fraud_details: {
          stripe_report?: "fraudulent";
          user_report?: UserReport;
        };
        livemode: boolean;
        metadata: {};
        order: string | null;
        shipping: unknown;
        billing_details: {
          address: {
            city: string | null;
            country: string | null;
            line1: string | null;
            line2: string | null;
            postal_code: string | null;
            state: string | null;
          };
          email: string | null;
          name: string | null;
          phone: string | null;
        };
        customer: string | null;
        payment_method: string;
        transfer_group: unknown;
        amount_refunded: number;
        refunded: boolean;
        review: string | null;
        created: number;
        balance_transaction: string | null;
        on_behalf_of: unknown;
        outcome: {
          seller_message: string;
          type: string;
          network_status: string;
          reason: string | null;
          risk_level: string;
          risk_score: number;
        };
        statement_descriptor: unknown;
        status: string;
        application: unknown;
        calculated_statement_descriptor: string;
        captured: boolean;
        failure_message: string | null;
        receipt_email: unknown;
        refunds: {
          total_count: number;
          url: string;
          object: string;
          data: Array<unknown>;
          has_more: boolean;
        };
        application_fee_amount: unknown;
        object: string;
        paid: boolean;
        payment_intent: unknown;
        id: string;
        currency: string;
        description: string;
        destination: unknown;
        failure_code: unknown;
        invoice: unknown;
        payment_method_details: {
          card: {
            checks: {
              address_line1_check: unknown;
              address_postal_code_check: unknown;
              cvc_check: unknown;
            };
            country: string;
            exp_month: number;
            last4: string;
            network: string;
            three_d_secure: unknown;
            brand: string;
            exp_year: number;
            fingerprint: string;
            funding: string;
            installments: unknown;
            wallet: unknown;
          };
          type: string;
        };
        source: {
          address_city: string | null;
          country: string;
          dynamic_last4: string | null;
          exp_month: number;
          funding: string;
          metadata: {};
          address_zip: string | null;
          customer: string | null;
          cvc_check: string | null;
          object: string;
          address_country: string | null;
          brand: string;
          exp_year: number;
          name: string | null;
          fingerprint: string;
          last4: string;
          id: string;
          address_line1: string | null;
          address_line1_check: string | null;
          address_line2: string | null;
          address_state: string | null;
          address_zip_check: string | null;
          tokenization_method: string | null;
        };
        application_fee: unknown;
      };
    };
    livemode: boolean;
    pending_webhooks: number;
    request: {
      id: string;
      idempotency_key: string;
    };
  };
  user: {
    email?: string;
  };
  v?: string;
  ts?: number;
};

export type EventTriggers = StripeChargeSucceeded;

export type Args = {
  event: EventTriggers;
  steps: {
    [clientID: string]: any;
  };
};
