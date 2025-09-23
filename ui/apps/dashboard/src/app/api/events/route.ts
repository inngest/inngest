import { NextResponse } from 'next/server';
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

export async function GET() {
  const { userId } = auth();
  if (!userId) {
    return NextResponse.json({ error: 'Please sign in to view events' }, { status: 401 });
  }

  return NextResponse.json(mockEventTypes);
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  // This is a placeholder for the actual event creation logic
  // In a real application, you would use the Events class to create the event
  // For now, we'll just return a success response
  return NextResponse.json({ message: 'Event created successfully', event: body });
}
