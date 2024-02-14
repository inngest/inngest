import React from "react";
import type { IconProps } from "./props";

const KeyboardGradient = ({ size = "56" }: IconProps) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 56 56"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <mask
      id="mask0_781_2836"
      style={{ maskType: "alpha" }}
      maskUnits="userSpaceOnUse"
      x="0"
      y="0"
      width="56"
      height="56"
    >
      <path
        d="M42 35L44.3333 35"
        stroke="white"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M21 35H35"
        stroke="white"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M14.0003 35H11.667"
        stroke="white"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M11.667 28H44.3337"
        stroke="white"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M11.667 21H44.3337"
        stroke="white"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M49.0003 14H7.00033C5.71166 14 4.66699 15.0447 4.66699 16.3333V39.6667C4.66699 40.9553 5.71166 42 7.00033 42H49.0003C50.289 42 51.3337 40.9553 51.3337 39.6667V16.3333C51.3337 15.0447 50.289 14 49.0003 14Z"
        stroke="white"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </mask>
    <g mask="url(#mask0_781_2836)">
      <rect width="56" height="56" fill="url(#paint0_linear_781_2836)" />
    </g>
    <defs>
      <linearGradient
        id="paint0_linear_781_2836"
        x1="0"
        y1="0"
        x2="56"
        y2="56"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#3C8CE7" />
        <stop offset="1" stopColor="#00EAFF" />
      </linearGradient>
    </defs>
  </svg>
);

export default KeyboardGradient;
