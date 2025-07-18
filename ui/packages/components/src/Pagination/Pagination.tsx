'use client';

import { useLayoutEffect, useMemo, useRef, useState, type SetStateAction } from 'react';
import {
  RiArrowLeftDoubleLine,
  RiArrowLeftSLine,
  RiArrowRightDoubleLine,
  RiArrowRightSLine,
  type RemixiconComponentType,
} from '@remixicon/react';

import { Button } from '../Button';
import { cn } from '../utils/classNames';
import { getVisiblePages } from './getVisiblePages';

// Both numbers and ellipses use same base styles to prevent shifting.
const PAGE_NUMBER_BASE_CLASSES =
  'flex h-8 items-center justify-center min-w-8 text-sm tabular-nums';

const NARROW_VARIANT_BREAKPOINT = 550;
const TINY_VARIANT_BREAKPOINT = 400;

export interface PaginationProps {
  currentPage: number;
  setCurrentPage: (action: SetStateAction<number>) => void;
  totalPages: number;
  variant?: 'normal' | 'narrow' | 'tiny';
  className?: string;
}

export function Pagination(props: PaginationProps) {
  const { className, currentPage, setCurrentPage, totalPages, variant: propVariant } = props;

  const outerRef = useRef<HTMLDivElement>(null);
  const [autoVariant, setAutoVariant] = useState<'narrow' | 'normal' | 'tiny'>('normal');

  useLayoutEffect(() => {
    if (propVariant !== undefined) return;

    const updateVariant = (width: number) => {
      if (width < TINY_VARIANT_BREAKPOINT) setAutoVariant('tiny');
      else if (width < NARROW_VARIANT_BREAKPOINT) setAutoVariant('narrow');
      else setAutoVariant('normal');
    };

    const observer = new ResizeObserver(([entry]) => {
      if (entry === undefined) return;
      updateVariant(entry.contentRect.width);
    });

    if (outerRef.current) {
      updateVariant(outerRef.current.getBoundingClientRect().width);
      observer.observe(outerRef.current);
    }

    return () => observer.disconnect();
  }, [propVariant]);

  const variant = propVariant ?? autoVariant;

  if (totalPages === 0) return null;

  const pages = useMemo(
    () => getVisiblePages({ current: currentPage, total: totalPages, variant }),
    [currentPage, totalPages, variant]
  );

  return (
    <div ref={outerRef} className={cn('flex w-full justify-center', className)}>
      <div className="flex items-center">
        <CaretButton {...props} typ="first" />
        <CaretButton {...props} typ="back" />

        <div className="flex gap-1">
          {pages.map((page, index) => {
            // Render ellipsis.
            if (typeof page !== 'number') {
              return (
                <div className={PAGE_NUMBER_BASE_CLASSES} key={`ellipsis-${index}`}>
                  {page}
                </div>
              );
            }

            const isActive = currentPage === page;

            return (
              <button
                key={page}
                onClick={() => setCurrentPage(page)}
                className={cn(
                  PAGE_NUMBER_BASE_CLASSES,
                  'rounded-md px-2',
                  isActive && 'bg-contrast text-onContrast',
                  !isActive && 'hover:bg-canvasSubtle'
                )}
              >
                {page}
              </button>
            );
          })}
        </div>

        <CaretButton {...props} typ="forward" />
        <CaretButton {...props} typ="last" />
      </div>
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
  const { currentPage, totalPages, setCurrentPage } = paginationProps;

  const onFirstPage = currentPage === 1;
  const onLastPage = currentPage === totalPages;

  let disabled = false;
  if (['back', 'first'].includes(typ) && onFirstPage) disabled = true;
  if (['forward', 'last'].includes(typ) && onLastPage) disabled = true;

  const Icon = CARET_ICON_MAP[typ];

  return (
    <Button
      appearance="ghost"
      className={cn(
        'group mx-1 h-8 w-8 rounded-md',
        !disabled && 'hover:bg-canvasSubtle',
        disabled && '!bg-transparent'
      )}
      disabled={disabled}
      icon={<Icon className="group-disabled:text-disabled text-basis h-6 w-6" />}
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
            setCurrentPage(paginationProps.totalPages);
            break;
        }
      }}
    />
  );
}
