import React from "react";
import type { IconProps } from "./props";

export default ({ size = "1em", fill = "currentColor" }: IconProps) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path
      d="M15 19.411L12.3 16.711L13.714 15.295L15 16.583L20.008 11.583L21.419 13L15 19.41V19.411ZM11 17H2V15H11V17ZM15 13H2V11H15V13ZM15 9H2V7H15V9Z"
      fill={fill}
    ></path>
  </svg>
);
