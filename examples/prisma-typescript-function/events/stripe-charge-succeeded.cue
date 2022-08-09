{
  // The unique name of the event
  name: "stripe/charge.succeeded"
  // The event payload, containing all event data
  data: {
    id:          string
    type:        "charge.succeeded"
    object:      string
    api_version: string
    created:     int
    data: {
      object: {
        amount_captured:             int
        receipt_number:              _
        receipt_url:                 string
        source_transfer:             _
        statement_descriptor_suffix: _
        transfer_data:               _
        amount:                      int
        dispute:                     _
        disputed:                    bool
        fraud_details: {
          stripe_report?: "fraudulent"
          user_report?:   "fraudulent" | "safe"
        }
        livemode: bool
        metadata: {}
        // The ID of the order for this charge, if one eixsts.
        order:    string | null
        shipping: _
        billing_details: {
          address: {
            city:        string | null
            country:     string | null
            line1:       string | null
            line2:       string | null
            postal_code: string | null
            state:       string | null
          }
          email: string | null
          name:  string | null
          phone: string | null
        }
        // The stripe ID of the customer for this charge, if one exists.
        customer:            string | null
        payment_method:      string
        transfer_group:      _
        amount_refunded:     int
        refunded:            bool
        review:              string | null
        created:             int
        balance_transaction: string | null
        on_behalf_of:        _
        outcome: {
          seller_message: string
          type:           string
          network_status: string
          reason:         string | null
          risk_level:     string
          risk_score:     int
        }
        statement_descriptor:            _
        status:                          string
        application:                     _
        calculated_statement_descriptor: string
        captured:                        bool
        // The error message explaining the reason for failure, if failed
        failure_message: string | null
        receipt_email:   _
        refunds: {
          total_count: int
          url:         string
          object:      string
          data: [...]
          has_more: bool
        }
        application_fee_amount: _
        object:                 string
        paid:                   bool
        payment_intent:         _
        id:                     string
        currency:               string
        description:            string
        destination:            _
        failure_code:           _
        invoice:                _
        payment_method_details: {
          card: {
            checks: {
              address_line1_check:       _
              address_postal_code_check: _
              cvc_check:                 _
            }
            country:        string
            exp_month:      int
            last4:          string
            network:        string
            three_d_secure: _
            brand:          string
            exp_year:       int
            fingerprint:    string
            funding:        string
            installments:   _
            wallet:         _
          }
          type: string
        }
        source: {
          address_city:  string | null
          country:       string
          dynamic_last4: string | null
          exp_month:     int
          funding:       string
          metadata: {}
          address_zip:         string | null
          customer:            string | null
          cvc_check:           string | null
          object:              string
          address_country:     string | null
          brand:               string
          exp_year:            int
          name:                string | null
          fingerprint:         string
          last4:               string
          id:                  string
          address_line1:       string | null
          address_line1_check: string | null
          address_line2:       string | null
          address_state:       string | null
          address_zip_check:   string | null
          tokenization_method: string | null
        }
        application_fee: _
      }
    }
    livemode:         bool
    pending_webhooks: int
    request: {
      id:              string
      idempotency_key: string
    }
  }
  // User information for the author of the event
  user: {
    email?: string
  }

  // An optional event version
  v?: string

  // The epoch of the event, in milliseconds
  ts?: number
}