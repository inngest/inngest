import React from "react";

type Props = {
  width?: string | number;
  height?: string | number;
  fill?: string;
};

export default ({ width, height, fill = "#111" }: Props) => (
  <svg
    viewBox="0 0 512 512"
    width={width || "32"}
    height={height || "32"}
    color={fill || "#000"}
  >
    <circle fill={fill} cx="256" cy="256" r="64" />
    <circle fill={fill} cx="256" cy="448" r="64" />
    <circle fill={fill} cx="256" cy="64" r="64" />
  </svg>
);
