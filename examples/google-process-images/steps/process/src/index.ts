import { Storage } from "@google-cloud/storage";
import fetch, { Response } from "node-fetch";
import sharp from "sharp";
import { Writable } from "stream";
import { ulid } from "ulid";
import type { Args } from "./types";

interface Thumbnail {
  size: number;
  uploadStream: Writable;
  pipeline: Writable;
}

const baseThumbnails: (Partial<Thumbnail> & Pick<Thumbnail, "size">)[] = [
  { size: 50 },
  { size: 100 },
  { size: 250 },
];

const gcs = new Storage({
  credentials: JSON.parse(process.env.GOOGLE_SERVICE_ACCOUNT as string),
});

const bucket = gcs.bucket("test-bucket");

export async function run({
  event: {
    data: { url },
  },
}: Args) {
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
    const uploadStream = bucket
      .file(`${ulid()}_${size}x${size}.png`)
      .createWriteStream({ metadata: { contentType: "image/png" } });
    const pipeline = sharp().resize(size).png().pipe(uploadStream);

    return {
      size,
      uploadStream,
      pipeline,
    };
  });

  /**
   * Start and wait for every thumbnail to be created.
   */
  const uploads = await Promise.all(
    thumbnails.map(
      ({ size, uploadStream, pipeline }) =>
        new Promise((resolve, reject) => {
          uploadStream.on("finish", resolve).on("error", reject);
          res.body!.pipe(pipeline);
        })
    )
  );
}
