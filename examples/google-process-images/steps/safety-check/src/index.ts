import vision from "@google-cloud/vision";
import type { Args } from "./types";

/**
 * Create a client for checking image safety using the Google Cloud Vision API.
 *
 * `process.env.GOOGLE_SERVICE_ACCOUNT` is a stringified JSON secret that you
 * would add to your Inngest Cloud account.
 *
 * With a JSON Google Service Account file, you could create a stringified
 * version with:
 *
 *     node -e 'console.log(JSON.stringify(require("./key.json")))' > secret
 *
 * @link https://www.inngest.com/docs/functions/secrets
 */
const client = new vision.ImageAnnotatorClient({
  credentials: JSON.parse(process.env.GOOGLE_SERVICE_ACCOUNT as string),
});

/**
 * The function to run for this step.
 */
export async function run({
  event: {
    data: { url },
  },
}: Args) {
  /**
   * Pass the `url` from the event to the Google Cloud Vision API to see if it
   * deems the image as safe.
   */
  const [result] = await client.safeSearchDetection(url);
  const detections = result.safeSearchAnnotation;

  /**
   * If we've failed to make any detections, fail; we need to ensure images are
   * safe, so we can't continue if we don't know.
   */
  if (!detections) {
    return {
      status: 500,
      error: "Could not make any detections for image",
      result,
    };
  }

  /**
   * Step up some thresholds for safety that we're looking to enforce.
   */
  const thresholds: typeof detections["adult"][] = [
    "POSSIBLE",
    "LIKELY",
    "VERY_LIKELY",
  ];

  /**
   * Decide whether the image is safe.
   */
  const isSafe =
    !thresholds.includes(detections.adult) &&
    !thresholds.includes(detections.violence);

  return {
    status: 200,
    body: {
      isSafe: isSafe,
      detections,
    },
  };
}
