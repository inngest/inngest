import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={20}
      height={18}
      viewBox="0 0 20 18"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        d="M20 14l-4 4-4-4h3V4h-4V2h5a1 1 0 011 1v11h3zM9 14v2H4a1 1 0 01-1-1V4H0l4-4 4 4H5v10h4z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

