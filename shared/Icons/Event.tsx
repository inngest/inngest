import React from "react";
import type { IconProps } from "./props";

const Event = ({
  size = "1em",
  fill = "currentColor",
  className,
  style,
}: IconProps) => {
  return (
    <svg
      height={size}
      width={size}
      className={className}
      viewBox="0 0 18 8"
      xmlns="http://www.w3.org/2000/svg"
      style={style}
    >
      <g fill={fill} fillRule="evenodd">
        <path d="M2.987 1.28c.3556 0 .6967.0583 1.013.1651-.5943.732-.9467 1.6449-.9467 2.6349s.3524 1.903.946 2.6345A3.1434 3.1434 0 0 1 2.987 6.88C1.3373 6.88 0 5.6264 0 4.08c0-1.5464 1.3373-2.8 2.987-2.8Z" />
        <circle cx="13.44" cy="4" r="4" />
        <path
          d="M7.9228.8c.3046 0 .599.0447.8772.128C8.0624 1.7306 7.6105 2.8112 7.6105 4s.4519 2.2694 1.1892 3.0714a3.0284 3.0284 0 0 1-.8769.1286C6.1981 7.2 4.8 5.7673 4.8 4S6.1981.8 7.9228.8Z"
          fillRule="nonzero"
        />
      </g>
    </svg>
  );
};

export default Event;
