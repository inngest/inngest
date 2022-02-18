import * as React from "react"

const History: React.FC<{}> = (props) => {
  return (
    <svg
      width={64}
      height={64}
      viewBox="0 0 64 64"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <rect width={64} height={64} rx={8} fill="#4636F5" />
      <path
        d="M20.693 34.963a4 4 0 116.702-3.829h9.224a3.995 3.995 0 11-.001 2h-9.222a4 4 0 01-6.703 1.829zm4.243-4.243a2 2 0 10-2.829 2.828 2 2 0 002.829-2.828zm16.97 0a2 2 0 10-.063 2.892l-.283.283.346-.347a2 2 0 000-2.828z"
        fill="#fff"
      />
    </svg>
  )
}

export default History

