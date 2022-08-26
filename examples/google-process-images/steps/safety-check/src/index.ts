import vision from "@google-cloud/vision";
import { google } from "@google-cloud/vision/build/protos/protos";
import type { Args } from "./types";

const client = new vision.ImageAnnotatorClient({
  credentials: JSON.parse(process.env.GOOGLE_SERVICE_ACCOUNT as string),
});

export async function run({
  event: {
    data: { url },
  },
}: Args) {
  const [result] = await client.safeSearchDetection(url);
  const detections = result.safeSearchAnnotation;

  if (!detections) {
    return {
      status: 500,
      error: "Could not make any detections for image",
      result,
    };
  }

  const thresholds: typeof detections["adult"][] = [
    "POSSIBLE",
    "LIKELY",
    "VERY_LIKELY",
  ];

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
