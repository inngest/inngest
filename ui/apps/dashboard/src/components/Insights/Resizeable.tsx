'use client';

import type { ReactNode } from 'react';

type ResizeableProps = {
  first: ReactNode;
  orientation: 'horizontal' | 'vertical';
  second: ReactNode;
};

export function Resizeable({ first, second }: ResizeableProps) {
  return (
    <>
      {first}
      {second}
    </>
  );
}
