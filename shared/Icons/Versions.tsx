import * as React from "react"

function SvgComponent(props) {
  return (
    <svg
      width={26}
      height={9}
      viewBox="0 0 26 9"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        d="M1.693 6.963a4 4 0 116.702-3.829h9.224a3.995 3.995 0 11-.001 2H8.396a4 4 0 01-6.703 1.829zM5.936 2.72a2 2 0 10-2.829 2.828A2 2 0 005.936 2.72zm16.97 0a2 2 0 10-.063 2.892l-.283.283.346-.347a2 2 0 000-2.828z"
        fill="#fff"
      />
    </svg>
  )
}

export default SvgComponent
