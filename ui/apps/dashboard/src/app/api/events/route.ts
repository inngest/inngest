import { NextRequest, NextResponse } from 'next/server';
import { auth } from '@clerk/nextjs/server';

// Temporary mock event types. Replace with real API call later.
const mockEventTypes: string[] = [
  'app/user.created',
  'app/user.updated',
  'app/user.disabled',
  'app/user.deleted',
  'session.started',
  'session.ended',
  'page.viewed',
  'purchase.completed',
  'purchase.refunded',
  'payment.failed',
  'payment.completed',
  'email.sent',
  'email.bounced',
  'feature.toggled_on',
  'feature.toggled_off',
  'error.logged',
];

export async function GET(_req: NextRequest) {
  const { userId } = auth();
  if (!userId) {
    return NextResponse.json({ error: 'Please sign in to view events' }, { status: 401 });
  }

  return NextResponse.json(mockEventTypes);
}
