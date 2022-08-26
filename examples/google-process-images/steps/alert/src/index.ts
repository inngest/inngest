import type { Args } from "./types";

/**
 * The function to run for this step.
 */
export async function run({
  event: {
    data: { url },
    user,
  },
}: Args) {
  /**
   * Here, we return that the user uploaded an unsafe image along with their
   * user details and the URL in question.
   *
   * In this area, you might want to notify moderators of your platform or flag
   * the account for review.
   */
  return {
    status: 200,
    body: {
      message: "User uploaded a potentially-unsafe image",
      url,
      user,
    },
  };
}
