import type { NextApiRequest, NextApiResponse } from "next";
import React from "react";
import * as ReactDOMServer from "react-dom/server";
import sharp from "sharp";

import SocialPreview from "../../shared/Docs/SocialPreview";

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  const { title } = req.query;
  const svg = ReactDOMServer.renderToString(
    <SocialPreview
      title={(Array.isArray(title) ? title[0] : title) || "Documentation"}
    />
  );

  const buffer = Buffer.from(svg);
  const body = await sharp(buffer)
    .resize({ width: 1200 })
    .png({ compressionLevel: 8, quality: 100 })
    .toBuffer();

  res.setHeader("Content-Type", "image/png");
  res.status(200).send(body);
}
