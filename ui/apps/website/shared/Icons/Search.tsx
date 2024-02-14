import React from "react";

export default ({ fill = "#222631", size = 15, style = {} }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 15 15"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    style={style}
  >
    <g>
      <mask
        id="mask0"
        mask-type="alpha"
        maskUnits="userSpaceOnUse"
        x="0"
        y="0"
        width="15"
        height="15"
      >
        <rect width="15" height="15" fill="white" />
      </mask>
      <g mask="url(#mask0)">
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M8.98521 7.79805C10.0289 6.33354 9.89383 4.28724 8.57993 2.97335C7.11547 1.50888 4.7411 1.50888 3.27663 2.97335C1.81217 4.43782 1.81217 6.81218 3.27663 8.27665C4.59052 9.59054 6.63683 9.72563 8.10133 8.68193L11.6735 12.2541L12.5574 11.3702L8.98521 7.79805C8.98521 7.79805 10.0289 6.33354 8.98521 7.79805ZM7.69605 3.85723C8.67236 4.83354 8.67236 6.41646 7.69605 7.39277C6.71974 8.36908 5.13683 8.36908 4.16052 7.39277C3.18421 6.41646 3.18421 4.83354 4.16052 3.85723C5.13683 2.88092 6.71974 2.88092 7.69605 3.85723C7.69605 3.85723 8.67236 4.83354 7.69605 3.85723Z"
          fill={fill}
        />
      </g>
    </g>
  </svg>
);
