import * as React from "react"

const Hub: React.FC<{}> = (props) => {
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
        d="M32 42c-4.561 0-8-1.935-8-4.5V26h.047a4.3 4.3 0 012.535-2.867A12.277 12.277 0 0132 22c4.561 0 8 1.935 8 4.5a2.805 2.805 0 01-.048.5H40v10.5c0 2.565-3.439 4.5-8 4.5zm-6.223-12.606V37.5c0 1.019 2.423 2.5 6.223 2.5s6.222-1.481 6.222-2.5v-8.106A11.3 11.3 0 0132 31a11.305 11.305 0 01-6.223-1.606zM32 24c-3.8 0-6.223 1.481-6.223 2.5S28.2 29 32 29s6.222-1.481 6.222-2.5S35.8 24 32 24z"
        fill="#fff"
      />
    </svg>
  )
}

export default Hub
