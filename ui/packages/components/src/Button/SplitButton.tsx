import React from 'react';

import { cn } from '../utils/classNames';

type SplitButtonProps = {
  left: React.ReactElement;
  right: React.ReactElement;
};

export const SplitButton = ({ left, right }: SplitButtonProps) => {
  return (
    <div className="flex flex-row items-center justify-center">
      {React.cloneElement(left, {
        className: cn(left.props.className, 'rounded-r-none'),
      })}
      {React.cloneElement(right, {
        className: cn(right.props.className, 'rounded-l-none border-l-0'),
      })}
    </div>
  );
};
