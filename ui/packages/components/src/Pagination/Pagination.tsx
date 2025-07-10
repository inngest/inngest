'use client';

import { useMemo, type Dispatch, type SetStateAction } from 'react';
import {
  RiArrowLeftDoubleLine,
  RiArrowLeftSLine,
  RiArrowRightDoubleLine,
  RiArrowRightSLine,
  type RemixiconComponentType,
} from '@remixicon/react';

import { Button } from '../Button';
import { cn } from '../utils/classNames';

interface PaginationProps {
  currentPage: number;
  numPages: number;
  setCurrentPage: Dispatch<SetStateAction<number>>;
}

export function Pagination(props: PaginationProps) {
  const { currentPage, numPages, setCurrentPage } = props;

  if (numPages === 0) return null;

  const pages = useMemo(
    () =>
      Array(numPages)
        .fill(null)
        .map((_, i) => i + 1),
    [numPages]
  );

  return (
    <div className="flex items-center">
      <CaretButton {...props} typ="first" />
      <CaretButton {...props} typ="back" />
      {pages.map((page) => {
        const isActive = currentPage === page;

        return (
          <button
            key={page}
            onClick={() => setCurrentPage(page)}
            className={cn(
              'mx-0 rounded-md px-3 py-1 text-sm',
              isActive && 'bg-contrast text-onContrast',
              !isActive && 'hover:bg-canvasSubtle'
            )}
          >
            {page}
          </button>
        );
      })}
      <CaretButton {...props} typ="forward" />
      <CaretButton {...props} typ="last" />
    </div>
  );
}

const CARET_ICON_MAP: Record<'back' | 'first' | 'forward' | 'last', RemixiconComponentType> = {
  back: RiArrowLeftSLine,
  first: RiArrowLeftDoubleLine,
  forward: RiArrowRightSLine,
  last: RiArrowRightDoubleLine,
};

interface CaretButtonProps extends PaginationProps {
  typ: keyof typeof CARET_ICON_MAP;
}

function CaretButton({ typ, ...paginationProps }: CaretButtonProps) {
  const { currentPage, numPages, setCurrentPage } = paginationProps;

  const onFirstPage = currentPage === 1;
  const onLastPage = currentPage === numPages;

  let disabled = false;
  if (['back', 'first'].includes(typ) && onFirstPage) disabled = true;
  if (['forward', 'last'].includes(typ) && onLastPage) disabled = true;

  const Icon = CARET_ICON_MAP[typ];

  return (
    <Button
      appearance="ghost"
      className="group mr-1 h-6 w-6 p-0"
      disabled={disabled}
      icon={<Icon className="bg-canvasBase group-disabled:text-disabled text-basis h-6 w-6" />}
      onClick={() => {
        switch (typ) {
          case 'back':
            setCurrentPage((p) => p - 1);
            break;
          case 'first':
            setCurrentPage(1);
            break;
          case 'forward':
            setCurrentPage((p) => p + 1);
            break;
          case 'last':
            setCurrentPage(paginationProps.numPages);
            break;
        }
      }}
    />
  );
}
