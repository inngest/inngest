import type { Args } from "./types";
import fetch from "node-fetch";

export async function run({ event }: Args) {
  try {
    const result = await fetch("https://slack.com/api/chat.postMessage", {
      method: "POST",
      headers: { authorization: `Bearer ${process.env.SLACK_TOKEN}` },
      body: JSON.stringify({
        channel: '#signups',
        blocks: [
          {
            type: 'section',
            text: { type: 'mrkdwn', text: 'New signup!' },
            fields: [
              { type: 'mrkdwn', text: `*Name*\n${event.user.email}` },
              { type: 'mrkdwn', text: `*Plan*\n$${event.data.plan_name}` },
            ]
          },
        ],
        username: 'Signup Bot',
        icon_emoji: ':tada:'
      }),
    });
    const body: any = await result.json();

    if (body.error !== undefined) {
      // Slack always returns a 200, even if there's an error.
      return { status: result.status === 200 ? 500 : result.status, body };
    }

    return { status: result.status, body };
  } catch(e) {
    return { status: 500, error: e }
  }

}
