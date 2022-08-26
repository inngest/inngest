import type { Args } from "./types";
import { Client } from "intercom-client";

export async function run({ event }: Args) {
  const client = new Client({ tokenAuth: { token: process.env.INTERCOM_TOKEN || "" } });

  try {
  const user = await client.contacts.createUser({
    externalId: event.user.external_id,
    email: event.user.email,
    // Intercom's epochs are in seconds whereas javascript uses milliseconds. 
    signedUpAt: event.ts / 1000,
    lastSeenAt: event.ts / 1000,
    isUnsubscribedFromEmails: !!!event.data.subscribed,
  });
  return { intercomID: user.id };
  } catch(e) {
    return { status: 500, error: e };
  }

}
