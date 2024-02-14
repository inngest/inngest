import React from "react";
import type { IconProps } from "./props";

const TrendingUp = ({ size = "1em", fill = "currentColor" }: IconProps) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 44 44"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path
      d="M36.6663 12.8335L23.1279 26.5835L17.4869 20.8543L7.33301 31.1668"
      stroke="url(#paint0_linear_781_2282)"
      strokeWidth="4"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M36.667 23.8335V12.8335H25.667"
      stroke="url(#paint1_linear_781_2282)"
      strokeWidth="4"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <defs>
      <linearGradient
        id="paint0_linear_781_2282"
        x1="21.9997"
        y1="12.8335"
        x2="21.9997"
        y2="31.1668"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#5D5FEF" />
        <stop offset="1" stopColor="#EF5DA8" />
      </linearGradient>
      <linearGradient
        id="paint1_linear_781_2282"
        x1="31.167"
        y1="12.8335"
        x2="31.167"
        y2="23.8335"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#5D5FEF" />
        <stop offset="1" stopColor="#EF5DA8" />
      </linearGradient>
    </defs>
  </svg>
);

export default TrendingUp;
