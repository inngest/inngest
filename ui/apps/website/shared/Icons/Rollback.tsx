import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={17}
      height={16}
      viewBox="0 0 17 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        d="M8.011 0a8 8 0 107.735 10h-2.08A6 6 0 118.01 2a5.92 5.92 0 014.223 1.78L9.016 7h7V0l-2.35 2.35A7.965 7.965 0 008.01 0z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

