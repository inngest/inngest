import React from "react";

const Icon = ({ size = 24, fill = "currentColor" }) => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke={fill}
    height={size}
    width={size}
  >
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
  </svg>
);
export default Icon;
