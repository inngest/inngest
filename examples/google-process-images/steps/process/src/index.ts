import { File, Storage } from "@google-cloud/storage";
import fetch, { Response } from "node-fetch";
import sharp from "sharp";
import { Writable } from "stream";
import { ulid } from "ulid";
import type { Args } from "./types";

/**
 * A representation of a thumbnail of a given `size` that will be uploaded to
 * Google Cloud Storage.
 */
interface Thumbnail {
  size: number;
  storageRef: File;
  uploadStream: Writable;
  resizer: Writable;
}

/**
 * The stubs for the thumbnails we'll be creating of the given image.
 */
const baseThumbnails: (Partial<Thumbnail> & Pick<Thumbnail, "size">)[] = [
  { size: 50 },
  { size: 100 },
  { size: 250 },
];

/**
 * Create clients for interacting with Google Cloud Storage.
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
const bucket = new Storage({
  credentials: JSON.parse(process.env.GOOGLE_SERVICE_ACCOUNT as string),
}).bucket("example_bucket_inngest");

/**
 * The function to run for this step.
 */
export async function run({
  event: {
    data: { url },
  },
}: Args) {
  /**
   * Try and fetch the image using the `url` from the event. Don't use the body
   * yet, but ensure it's a successful response before continuing.
   */
  const res = await new Promise<Response>(async (resolve, reject) => {
    fetch(url, {
      redirect: "follow",
    })
      .then((res) => {
        if (res.status !== 200) {
          throw new Error(
            `Error downloading image; invalid response status code: ${res.status}`
          );
        }

        resolve(res);
      })
      .catch(reject);
  });

  if (!res.body) {
    throw new Error("No body to pipe");
  }

  /**
   * For every size, let's create an upload stream and a processor ready to pipe
   * the request through.
   */
  const thumbnails: Thumbnail[] = baseThumbnails.map(({ size }) => {
    // The location we'll save the file in our bucket.
    const storageRef = bucket.file(`${ulid()}_${size}x${size}.png`);

    // An writable stream to Google Cloud Storage, ready to send our thumbnail
    const uploadStream = storageRef.createWriteStream({
      metadata: { contentType: "image/png" },
    });

    // Use Sharp to resize our streamed image on the fly
    const resizer = sharp()
      .resize({
        width: size,
        height: size,
        fit: sharp.fit.cover,
        position: sharp.strategy.entropy,
      })
      .png();

    return {
      size,
      storageRef,
      uploadStream,
      resizer,
    };
  });

  /**
   * Start and wait for every thumbnail to be created.
   */
  const uploads = await Promise.all(
    thumbnails.map(
      ({ uploadStream, resizer, storageRef }) =>
        new Promise<string>((resolve, reject) => {
          resizer.on("error", reject);

          uploadStream
            .on("finish", () => resolve(storageRef.publicUrl()))
            .on("error", reject);

          res.body!.pipe(resizer).pipe(uploadStream);
        })
    )
  );

  return {
    status: 200,
    body: {
      uploads,
    },
  };
}
