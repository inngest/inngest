import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={16}
      height={20}
      viewBox="0 0 16 20"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        d="M8 20a2 2 0 01-2-2h4a2 2 0 01-2 2zm8-3H0v-2l2-1V8.5c0-3.462 1.421-5.707 4-6.32V0h4v2.18c2.579.612 4 2.856 4 6.32V14l2 1v2zM8 3.75A3.6 3.6 0 004.875 5.2 5.692 5.692 0 004 8.5V15h8V8.5a5.693 5.693 0 00-.875-3.3A3.6 3.6 0 008 3.75z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

