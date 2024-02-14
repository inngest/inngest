import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={18}
      height={12}
      viewBox="0 0 18 12"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path d="M17.5 12L9 6l8.5-6v12zm-9 0L0 6l8.5-6v12z" fill="#fff" />
    </svg>
  )
}

export default SvgComponent

