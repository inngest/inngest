import React from 'react';

import type { IconProps } from './props';

export default ({ size = '1em', fill = 'currentColor' }: IconProps) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path d="M13 13V19H11V13H5V11H11V5H13V11H19V13H13Z" fill={fill}></path>
  </svg>
);
