import React from "react";
import type { IconProps } from "./props";

const FileCheck = ({ size = "1em", fill = "currentColor" }: IconProps) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path
      d="M15.7071 13.7071C16.0976 13.3166 16.0976 12.6834 15.7071 12.2929C15.3166 11.9024 14.6834 11.9024 14.2929 12.2929L15.7071 13.7071ZM11 17L10.2929 17.7071C10.6834 18.0976 11.3166 18.0976 11.7071 17.7071L11 17ZM9.70711 14.2929C9.31658 13.9024 8.68342 13.9024 8.29289 14.2929C7.90237 14.6834 7.90237 15.3166 8.29289 15.7071L9.70711 14.2929ZM14.2929 12.2929L10.2929 16.2929L11.7071 17.7071L15.7071 13.7071L14.2929 12.2929ZM11.7071 16.2929L9.70711 14.2929L8.29289 15.7071L10.2929 17.7071L11.7071 16.2929Z"
      fill={fill}
    />
    <path
      d="M18 21H6C5.44772 21 5 20.5523 5 20L5 4C5 3.44772 5.44772 3 6 3L13.5631 3C13.8416 3 14.1076 3.11619 14.2968 3.32059L18.7338 8.11246C18.9049 8.29731 19 8.53995 19 8.79187L19 20C19 20.5523 18.5523 21 18 21Z"
      stroke={fill}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M19 9L14 9C13.4477 9 13 8.55228 13 8L13 3"
      stroke={fill}
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

export default FileCheck;
