import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={24}
      height={14}
      viewBox="0 0 24 14"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        d="M0 7.786l5.655 5.655 1.41-1.42-5.65-5.65L0 7.786zM21.915.002l-10.6 10.609-4.237-4.247-1.43 1.41 5.667 5.667 12.02-12.02-1.42-1.42zM17.675 1.42L16.266 0l-6.37 6.37 1.42 1.41 6.36-6.36z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

