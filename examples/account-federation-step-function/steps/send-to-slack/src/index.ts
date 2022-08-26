import type { Args } from "./types";
import fetch from "node-fetch";

export async function run({ event }: Args) {
  try {
  await fetch("https://slack.com/api/chat.postMessage", {
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
  } catch(e) {
    return { status: 500, error: e }
  }

  return { status: 200 };
}
