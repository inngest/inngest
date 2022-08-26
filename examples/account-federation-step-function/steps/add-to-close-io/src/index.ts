import type { Args } from "./types";
import fetch from "node-fetch";

export async function run({ event }: Args) {

  const auth = btoa(`${process.env.CLOSE_API_KEY}:`);

  try {
    const result = await fetch("https://api.close.com/api/v1/lead", {
      method: "POST",
      headers: {
        authorization: `Basic ${auth}`,
        "content-type": "application/json",
      },
      body: JSON.stringify({
        "name": event.user.email,
        "contacts": [
          {
            "name": event.user.email,
            "emails": [
              {
                "type": "office",
                "email": event.user.email
              }
            ],
          }
        ],
      }),
    });

    const body: any = await result.json();

    if (body.error !== undefined) {
      return { status: result.status, error: body.error }
    }

    return { status: result.status, body }
  } catch (e) {
    return { status: 500, error: e, auth };
  }
}
