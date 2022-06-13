import React, { ReactElement } from "react";
import * as ReactDOMServer from "react-dom/server";
import sharp from "sharp";

export default async function renderReactToPng({
  Component,
  props,
}: {
  Component: React.FC;
  props: any;
}) {
  const markupString = ReactDOMServer.renderToString(
    React.createElement(Component, props)
  );

  const buffer = Buffer.from(markupString);
  const body = await sharp(buffer)
    .resize({ width: 1200 })
    .png({ compressionLevel: 8, quality: 100 })
    .toBuffer();
  return body;
}
