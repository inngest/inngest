import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={16}
      height={16}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        d="M16 16h-5.5l2.049-2.05-3.129-3.13 1.41-1.41 3.13 3.129L16 10.5V16zM1.41 16L0 14.59 12.54 2.04 10.5 0H16v5.5l-2.04-2.04L1.411 16H1.41zm3.76-9.42L0 1.41 1.41 0l5.18 5.17-1.42 1.409v.001z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

