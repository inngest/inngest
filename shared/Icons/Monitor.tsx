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
        d="M22.222 24H9.778A1.778 1.778 0 018 22.222V9.778C8 8.796 8.796 8 9.778 8h12.444C23.204 8 24 8.796 24 9.778v12.444c0 .982-.796 1.778-1.778 1.778zM9.778 9.778v12.444h12.444V9.778H9.778zm10.666 10.666h-1.777v-6.222h1.777v6.222zm-3.555 0H15.11v-8.888h1.778v8.888zm-3.556 0h-1.777V16h1.777v4.444z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent

