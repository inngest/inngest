import React from "react";

type Props = {
  width?: string | number;
  height?: string | number;
  color?: string;
  stroke?: string | number;
};

export default ({ width, height, color, stroke }: Props) => (
  <svg
    role="img"
    xmlns="http://www.w3.org/2000/svg"
    width={width || "64"}
    height={height || "64"}
    viewBox="0 0 24 24"
    stroke={color || "#000"}
    strokeWidth={stroke || "2"}
    strokeLinecap="square"
    strokeLinejoin="miter"
    fill="none"
    color={color || "#000"}
  >
    <polyline points="7 13 10 16 17 9" /> <circle cx="12" cy="12" r="10" />
  </svg>
);
