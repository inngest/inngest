import type { Args } from "./types";
import vision from "@google-cloud/vision";
import { google } from "@google-cloud/vision/build/protos/protos";

const client = new vision.ImageAnnotatorClient({
  credentials: JSON.parse(process.env.GOOGLE_SERVICE_ACCOUNT as string),
});

export async function run({
  event: {
    data: { url },
    user: { email },
  },
}: Args) {
  const [result] = await client.safeSearchDetection(url);
  const detections = result.safeSearchAnnotation;

  console.log("detections:", detections, result);

  if (!detections) {
    throw new Error("Could not make any detections for image");
  }

  console.log(`Detections for image "${url}" for user "${email}":`, detections);

  const thresholds: typeof detections["adult"][] = [
    google.cloud.vision.v1.Likelihood.POSSIBLE,
    google.cloud.vision.v1.Likelihood.LIKELY,
    google.cloud.vision.v1.Likelihood.VERY_LIKELY,
  ];

  const isSafe =
    !thresholds.includes(detections.adult) &&
    !thresholds.includes(detections.violence);

  console.log("isSafe:", isSafe);

  return {
    status: 200,
    body: {
      isSafe: isSafe,
    },
  };
}
