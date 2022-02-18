import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={18}
      height={15}
      viewBox="0 0 18 15"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        d="M5 .5h13v2H5v-2zM0 0h3v3H0V0zm0 6h3v3H0V6zm0 6h3v3H0v-3zm5-5.5h13v2H5v-2zm0 6h13v2H5v-2z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

