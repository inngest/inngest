import React from "react";

export default ({ fill = "#222631", size = 24 }) => (
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
        d="M4.29289 10.7071C4.10536 10.8946 4 11.149 4 11.4142V21C4 21.5523 4.44772 22 5 22H19C19.5523 22 20 21.5523 20 21V11.4142C20 11.149 19.8946 10.8946 19.7071 10.7071L12.7071 3.70708C12.3166 3.31655 11.6834 3.31655 11.2929 3.70708L4.29289 10.7071C4.29289 10.7071 4.10536 10.8946 4.29289 10.7071ZM18 11.8284L12 5.8284L6 11.8284V20H10V15H14V20H18V11.8284Z"
        fill={fill}
      />
    </g>
  </svg>
);
