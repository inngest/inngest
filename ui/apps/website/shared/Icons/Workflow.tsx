import React from "react";

export default ({ fill = "#ffffff", size = 15, className = ""  }) => (
  <svg
    className={className} 
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
        d="M6.29608 9.89768L5.34774 10.0526C3.44924 10.3628 2 12.0141 2 14C2 16.2091 3.79086 18 6 18H19C20.6569 18 22 16.6569 22 15C22 13.5005 20.898 12.2548 19.4601 12.0348L18.1442 11.8334L17.822 10.5418C17.1711 7.93171 14.8087 6 12 6C9.75746 6 7.79944 7.22938 6.76775 9.06051L6.29608 9.89768ZM19.7626 10.0578C18.8948 6.57805 15.7485 4 12 4C9.00647 4 6.39696 5.64419 5.02529 8.07877C2.17514 8.54439 0 11.0182 0 14C0 17.3137 2.68629 20 6 20H19C21.7614 20 24 17.7614 24 15C24 12.4979 22.1621 10.425 19.7626 10.0578C19.7626 10.0578 18.8948 6.57805 19.7626 10.0578Z"
        fill={fill}
      />
    </g>
  </svg>
);
