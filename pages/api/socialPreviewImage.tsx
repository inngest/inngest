import type { NextApiRequest, NextApiResponse } from "next";
import React from "react";
import renderReactToPng from "../../utils/renderReactToPng";

import SocialPreview from "../../shared/Docs/SocialPreview";

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  const { title } = req.query;

  const body = await renderReactToPng({
    Component: SocialPreview,
    props: {
      title: (Array.isArray(title) ? title[0] : title) || "Documentation",
    },
  });

  res.setHeader("Content-Type", "image/png");
  res.status(200).send(body);
}
