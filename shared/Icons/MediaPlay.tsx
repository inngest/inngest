import React from "react";
import type { IconProps } from "./props";

const Github = ({ size = "1em", fill = "currentColor" }: IconProps) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={size}
    height={size}
    viewBox="0 0 24 24"
  >
    <path
      d="M20 12L5 4V20L20 12Z"
      stroke={fill}
      fill="transparent"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

export default Github;
