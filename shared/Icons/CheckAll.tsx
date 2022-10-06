import React from "react";
import type { IconProps } from "./props";

const CheckAll = ({ size = "1em", fill = "currentColor", className }: IconProps) => (
  <svg
    className={className}
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path
      d="M8 12.4853L12.2426 16.728L20.7279 8.24271"
      stroke={fill}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M3 12.4853L7.24264 16.728M12.5 11.5001L15.7279 8.24271"
      stroke={fill}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

export default CheckAll;
