import React from "react";

export default ({ fill = "#fff", size = 15 }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <mask
      id="mask0"
      mask-type="alpha"
      maskUnits="userSpaceOnUse"
      x="0"
      y="0"
      width="24"
      height="24"
    >
      <rect width="24" height="24" fill="white" />
    </mask>
    <g mask="url(#mask0)">
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M4 4C2.897 4 2 4.897 2 6V18C2 19.103 2.897 20 4 20H20C21.103 20 22 19.103 22 18V6C22 4.897 21.103 4 20 4H4ZM4 6H20L20.002 18H4V6ZM18 14V16H12V14H18ZM8.293 12.293L6 14.586L7.414 16L11.121 12.293L7.414 8.586L6 10L8.293 12.293Z"
        fill={fill}
      />
    </g>
  </svg>
);
