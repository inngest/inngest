import type { Args } from "./types";

export async function run({ event }: Args) {
  return { status: 200, body: `function ran from ${event.name}` };
}
