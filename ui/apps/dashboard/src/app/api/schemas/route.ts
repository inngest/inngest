import { NextRequest, NextResponse } from 'next/server';
import { auth } from '@clerk/nextjs/server';

// Temporary mock schemas. Replace with real API call later.
const mockSchemas: Record<string, unknown> = {
  'app/user.created': {
    event_name: 'app/user.created',
    timestamp: 'DateTime',
    user_id: 'String',
    email: 'String',
    plan: 'String',
    referrer: 'String | Null',
  },
  'app/user.updated': {
    event_name: 'app/user.updated',
    timestamp: 'DateTime',
    user_id: 'String',
    changed_fields: 'Array(String)',
  },
  'app/user.disabled': {
    event_name: 'app/user.disabled',
    timestamp: 'DateTime',
    user_id: 'String',
    reason: 'String | Null',
  },
  'app/user.deleted': {
    event_name: 'app/user.deleted',
    timestamp: 'DateTime',
    user_id: 'String',
    hard_delete: 'Bool',
  },
  'session.started': {
    event_name: 'session.started',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    session_id: 'String',
    device: 'String | Null',
  },
  'session.ended': {
    event_name: 'session.ended',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    session_id: 'String',
    duration_ms: 'UInt64',
  },
  'page.viewed': {
    event_name: 'page.viewed',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    url: 'String',
    referrer: 'String | Null',
  },
  'purchase.completed': {
    event_name: 'purchase.completed',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    order_id: 'String',
    amount_cents: 'UInt64',
    currency: 'String',
  },
  'purchase.refunded': {
    event_name: 'purchase.refunded',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    order_id: 'String',
    amount_cents: 'UInt64',
    reason: 'String | Null',
  },
  'payment.failed': {
    event_name: 'payment.failed',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    attempt_id: 'String',
    code: 'String',
    message: 'String | Null',
  },
  'payment.completed': {
    event_name: 'payment.completed',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    payment_id: 'String',
    amount_cents: 'UInt64',
    method: 'String',
  },
  'email.sent': {
    event_name: 'email.sent',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    email_id: 'String',
    template: 'String | Null',
  },
  'email.bounced': {
    event_name: 'email.bounced',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    email_id: 'String',
    bounce_type: 'String',
  },
  'feature.toggled_on': {
    event_name: 'feature.toggled_on',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    feature_key: 'String',
  },
  'feature.toggled_off': {
    event_name: 'feature.toggled_off',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    feature_key: 'String',
  },
  'error.logged': {
    event_name: 'error.logged',
    timestamp: 'DateTime',
    user_id: 'String | Null',
    error_class: 'String',
    message: 'String | Null',
    stack_present: 'Bool',
  },
};

export async function GET(_req: NextRequest) {
  const { userId } = auth();
  if (!userId) {
    return NextResponse.json({ error: 'Please sign in to view schemas' }, { status: 401 });
  }

  return NextResponse.json(mockSchemas);
}
