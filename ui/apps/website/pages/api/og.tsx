import type { NextApiRequest } from "next";
import { ImageResponse } from "@vercel/og";
import Logo from "src/shared/Icons/Logo";

export const config = {
  runtime: "experimental-edge",
};

const SVGBackgroundGradient =
  `<svg width="1200" height="630" viewBox="0 0 1200 630" fill="none" xmlns="http://www.w3.org/2000/svg">
<rect width="1200" height="630" fill="#0A0A12"/>
<rect width="1200" height="630" fill="url(#paint0_radial_1_2)"/>
<defs>
<radialGradient id="paint0_radial_1_2" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(273.5 281) rotate(47.4717) scale(1770.83 3373.01)">
<stop stop-color="#13123B"/>
<stop offset="1" stop-color="#08090D"/>
</radialGradient>
</defs>
</svg>
`.replace(/\n/g, "");

const font = fetch(
  new URL("../../public/assets/fonts/Inter/Inter-Medium.ttf", import.meta.url)
).then((res) => res.arrayBuffer());

export default async function handler(req: NextApiRequest) {
  try {
    const fontData = await font;
    const { searchParams } = new URL(req.url);

    // ?title=<title>
    const hasTitle = searchParams.has("title");
    const title = hasTitle
      ? searchParams.get("title")?.slice(0, 100)
      : "Inngest";
    const isLongTitle = title.length > 40;

    return new ImageResponse(
      (
        <div
          tw="border-t-8 border[#5d5fef]"
          style={{
            backgroundColor: "#0a0a12",
            backgroundImage: `url("data:image/svg+xml;utf8,${SVGBackgroundGradient}")`,
            height: "100%",
            width: "100%",
            padding: "7% 6%",
            display: "flex",
            alignItems: "flex-start",
            justifyContent: "flex-start",
            flexDirection: "column",
            flexWrap: "nowrap",
            fontFamily: "Inter, sans-serif",
          }}
        >
          <Logo
            width={160}
            fill={"#ffffff"}
            style={{
              display: "flex",
            }}
          />
          <div
            style={{
              fontSize: isLongTitle ? 84 : 110,
              fontStyle: "normal",
              fontWeight: 500,
              letterSpacing: "-2.4px",
              color: "white",
              marginTop: 48,
              padding: "0",
              lineHeight: 1.4,
              whiteSpace: "normal",
            }}
          >
            {title}
          </div>
        </div>
      ),
      {
        width: 1200,
        height: 630,
        fonts: [
          {
            name: "Inter",
            data: fontData,
            style: "normal",
            weight: 500,
          },
        ],
      }
    );
  } catch (e: any) {
    console.log(`${e.message}`);
    return new Response(`Failed to generate the image`, {
      status: 500,
    });
  }
}
