import fetch from "cross-fetch";
import { z } from "zod";

/**
 * Fetches a resource using `fetch` (via `cross-fetch`) and validates the output
 * using `zod`.
 *
 * Will throw if the request fails or zod validation fails, else will return a
 * typed copy of the data requested.
 *
 * @link https://npm.im/cross-fetch
 * @link https://npm.im/zod
 */
export const reqWithSchema = async <T extends z.ZodTypeAny>(
  /**
   * The request (or URL) to pass to `fetch`. Can be a `URL`, `Request`, or a
   * `string`.
   */
  req: Parameters<typeof fetch>[0],

  /**
   * The `zod` schema to use to validate the response.
   *
   * @link https://npm.im/zod
   */
  schema: T,

  /**
   * THIS SHOULD ONLY BE USED DURING BUILD STEPS.
   *
   * If provided, this small object will be used to cache incoming responses in
   * memory.
   */
  cache?: Record<string, any>
): Promise<z.output<T>> => {
  const url =
    typeof req === "string"
      ? req
      : Object.prototype.hasOwnProperty.call(req, "href")
      ? (req as unknown as URL).href
      : (req as Request).url;

  const json =
    cache?.[url] ||
    (await fetch(req, {
      headers: {
        "User-Agent": "inngest",

        /**
         * If a `GITHUB_TOKEN` env var is available, use it here in order to
         * increase the rate limit allowance.
         *
         * GitHub Action runners get 1,000 requests per hour. A single deploy
         * here uses a maximum of (3 + 3n) requests, where n is the number of
         * examples.
         *
         * e.g. a single deploy of 3 examples can use up to a total of 9
         * requests.
         *
         * Using a set token here allows us up to 15,000 requests per hour. If
         * we hit that limit, we can add a pre-build step to clone the repo and
         * change how this data is fetched.
         */
        Authorization: process.env.GITHUB_TOKEN
          ? `token ${process.env.GITHUB_TOKEN}`
          : "",
      },
    })
      .then((res) => res.json())
      .then((data) => {
        if (cache) cache[url] = data;
        return data;
      }));

  try {
    return schema.parse(json);
  } catch (err) {
    console.error("Error reading json:", json, err);
    throw err;
  }
};
