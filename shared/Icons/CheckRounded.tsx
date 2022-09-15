import React from "react";
import type { IconProps } from "./props";

type Props = IconProps & {
  shadow?: boolean;
};

export default ({ size = 32, fill = "currentColor", shadow = true }: Props) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 64 64"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g clipPath="url(#clip0_838_488)">
      <g filter={shadow ? "url(#filter0_d_838_488)" : ""}>
        <path
          d="M11.2734 31.6367L24.3228 44.686L52.2856 16.7231"
          stroke={fill}
          strokeWidth="6"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </g>
    </g>
    <defs>
      <filter
        id="filter0_d_838_488"
        x="-11.7266"
        y="-4.27686"
        width="87.0117"
        height="73.9629"
        filterUnits="userSpaceOnUse"
        colorInterpolationFilters="sRGB"
      >
        <feFlood floodOpacity="0" result="BackgroundImageFix" />
        <feColorMatrix
          in="SourceAlpha"
          type="matrix"
          values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
          result="hardAlpha"
        />
        <feOffset dy="2" />
        <feGaussianBlur stdDeviation="10" />
        <feComposite in2="hardAlpha" operator="out" />
        <feColorMatrix
          type="matrix"
          values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0"
        />
        <feBlend
          mode="normal"
          in2="BackgroundImageFix"
          result="effect1_dropShadow_838_488"
        />
        <feBlend
          mode="normal"
          in="SourceGraphic"
          in2="effect1_dropShadow_838_488"
          result="shape"
        />
      </filter>
      <clipPath id="clip0_838_488">
        <rect
          width="63.2727"
          height="63.2727"
          fill="white"
          transform="translate(0.727539)"
        />
      </clipPath>
    </defs>
  </svg>
);
