//
// Server-only: reads INNGEST_SIGNING_KEY. Never import this from client code.
//
// Every tenant's Insights chats run through one shared Inngest app, so the
// signing key scopes a run lookup to that app and nothing finer. Ownership has
// to be enforced by us.
//
// /api/chat stamps userId into the event from the Clerk session (never from the
// request body) and hands the browser a receipt for the event id it created.
// /api/chat-result only reads runs for an event whose receipt matches the
// caller's own session, so an event is only ever readable by the user who
// triggered it. A receipt is useless to anyone else: verification re-derives it
// from the session userId, not from the request.
import { createHmac, timingSafeEqual } from 'node:crypto';

function secret(): string {
  // Falls back to a per-process value so local dev (no signing key) still
  // round-trips its own receipts instead of trusting everything.
  return process.env.INNGEST_SIGNING_KEY || PROCESS_SECRET;
}

const PROCESS_SECRET = createHmac('sha256', 'insights-chat')
  .update(String(process.pid))
  .digest('hex');

export function signReceipt(eventId: string, userId: string): string {
  return createHmac('sha256', secret())
    .update(`${eventId}\n${userId}`)
    .digest('base64url');
}

export function verifyReceipt(
  eventId: string,
  userId: string,
  receipt: string,
): boolean {
  const expected = Buffer.from(signReceipt(eventId, userId));
  const given = Buffer.from(receipt);
  return expected.length === given.length && timingSafeEqual(expected, given);
}
