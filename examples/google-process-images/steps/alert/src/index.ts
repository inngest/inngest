import type { Args } from "./types";

export async function run({
  event: {
    data: { url },
    user,
  },
}: Args) {
  return {
    status: 200,
    body: {
      message: "User uploaded a potentially-unsafe image",
      url,
      user,
    },
  };
}
