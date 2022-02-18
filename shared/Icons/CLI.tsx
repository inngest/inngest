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
        d="M18 18H2a2 2 0 01-2-2V2a2 2 0 012-2h16a2 2 0 012 2v14a2 2 0 01-2 2zM2 4v12h16V4H2zm14 10h-6v-2h6v2zM5.414 14L4 12.586l2.293-2.292L4 8l1.414-1.414 3.706 3.707L5.415 14h-.001z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

