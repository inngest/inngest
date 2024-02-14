import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={32}
      height={32}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <rect width={32} height={32} rx={8} fill="#4636F5" />
      <path
        d="M11.201 24a3.202 3.202 0 111.627-5.96l5.22-5.22a3.198 3.198 0 111.132 1.132l-5.22 5.22A3.201 3.201 0 0111.201 24zm0-4.802a1.6 1.6 0 100 3.201 1.6 1.6 0 000-3.201zm9.605-9.606a1.6 1.6 0 101.601 1.673v.32-.392a1.6 1.6 0 00-1.6-1.6z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

