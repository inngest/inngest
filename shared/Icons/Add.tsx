import React from "react";

type Props = {
  size?: string | number;
  width?: string;
  height?: string;
  fill?: string;
};

export default ({ size, width, height, fill }: Props) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    x="0px"
    y="0px"
    width={size || width || "64"}
    height={size || height || "64"}
    stroke={fill || "#000"}
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    viewBox="0 0 24 24"
  >
    <path d="M12 5v14M5 12h14" />
  </svg>
);
