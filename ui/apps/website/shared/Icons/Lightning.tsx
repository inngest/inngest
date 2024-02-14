import React from "react";

type Props = {
  size?: string | number;
  color?: string;
  stroke?: string | number;
};

export default ({ size, color, stroke }: Props) => (
  <svg
    role="img"
    xmlns="http://www.w3.org/2000/svg"
    width={size || "24"}
    height={size || "24"}
    viewBox="0 0 24 24"
    stroke={color || "currentColor"}
    strokeWidth={stroke || "2"}
    strokeLinecap="square"
    strokeLinejoin="miter"
    fill="none"
    color={color || "currentColor"}
  >
  <path d="M13 10V3L4 14h7v7l9-11h-7z" />
</svg>
);

