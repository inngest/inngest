import React from "react";
import type { IconProps } from "./props";

const Archive = ({ size = "1em", fill = "currentColor" }: IconProps) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path
      d="M18 9H6C5.44772 9 5 9.44772 5 10V18C5 18.5523 5.44772 19 6 19H18C18.5523 19 19 18.5523 19 18V10C19 9.44772 18.5523 9 18 9Z"
      stroke={fill}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M10 14H14"
      stroke={fill}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M19.274 6H4.72594C4.40909 6 4.11098 6.15016 3.92238 6.40477L3.18164 7.40477C2.69278 8.06474 3.16389 9 3.9852 9H20.0148C20.8361 9 21.3072 8.06474 20.8183 7.40477L20.0776 6.40477C19.889 6.15016 19.5909 6 19.274 6Z"
      stroke={fill}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

export default Archive;
