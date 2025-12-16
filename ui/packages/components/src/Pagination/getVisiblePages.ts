const FIRST_PAGE_SLOT_SIZE_1 = 1;
const CURRENT_PAGE_SLOT_SIZE_1 = 1;
const LAST_PAGE_SLOT_SIZE_1 = 1;
const SINGLE_ELLIPSIS_SLOT_SIZE_1 = 1;

const EARLY_TRANSITION_OFFSET = 1;
const ONE_BASED_INDEX_OFFSET = 1;

type GetVisiblePagesConfig = {
  current: number;
  total: number;
  variant?: 'normal' | 'narrow' | 'tiny';
};

type PaginationConfig = {
  current: number;
  beginningConsecutivePages: number;
  endConsecutivePages: number;
  middleConsecutivePages: number;
  total: number;
};

export function getVisiblePages({
  current,
  total,
  variant = 'normal',
}: GetVisiblePagesConfig): Array<number | '...'> {
  if (variant === 'tiny') return [current];

  const slots = variant === 'narrow' ? 5 : 7;

  // How many consecutive pages each pattern shows (calculated to fill exactly <slots> slots)
  const beginningConsecutivePages = slots - SINGLE_ELLIPSIS_SLOT_SIZE_1 - LAST_PAGE_SLOT_SIZE_1;
  const endConsecutivePages = slots - FIRST_PAGE_SLOT_SIZE_1 - SINGLE_ELLIPSIS_SLOT_SIZE_1;
  const middleConsecutivePages =
    slots - FIRST_PAGE_SLOT_SIZE_1 - 2 * SINGLE_ELLIPSIS_SLOT_SIZE_1 - LAST_PAGE_SLOT_SIZE_1;

  if (slots >= total) return makeInclusiveRange(1, total);

  // Determine when to switch between pagination patterns to keep current page position stable.
  // The easiest way to understand EARLY_TRANSITION_OFFSET is to toggle it and observe the behavior difference on transitioning between patterns.
  const lastPageForBeginningPattern = beginningConsecutivePages - EARLY_TRANSITION_OFFSET;
  const firstPageForEndPattern =
    total - endConsecutivePages + ONE_BASED_INDEX_OFFSET + EARLY_TRANSITION_OFFSET;

  const config: PaginationConfig = {
    current,
    beginningConsecutivePages,
    endConsecutivePages,
    middleConsecutivePages,
    total,
  };

  if (current <= lastPageForBeginningPattern) return handleBeginningCase(config);
  else if (current >= firstPageForEndPattern) return handleEndCase(config);
  else return handleMiddleCase(config);
}

// Shows: [1, 2, 3, ..., N] where 3 is beginningConsecutivePages and N is total
function handleBeginningCase({
  beginningConsecutivePages,
  total,
}: PaginationConfig): Array<number | '...'> {
  return [...makeInclusiveRange(1, beginningConsecutivePages), '...', total];
}

// Shows: [1, ..., 8, 9, N] where 8 is startOfEndPages and N is total
function handleEndCase({ endConsecutivePages, total }: PaginationConfig): Array<number | '...'> {
  const startOfEndPages = total - endConsecutivePages + ONE_BASED_INDEX_OFFSET;
  return [1, '...', ...makeInclusiveRange(startOfEndPages, total)];
}

// Shows: [1, ..., 4, 5, 6, ..., N] where 5 is current and N is total
function handleMiddleCase({
  current,
  middleConsecutivePages,
  total,
}: PaginationConfig): Array<number | '...'> {
  // NOTE: Watch out for unequal divisions if adding more variants.
  const pagesBeforeCurrent = (middleConsecutivePages - CURRENT_PAGE_SLOT_SIZE_1) / 2;
  const pagesAfterCurrent = middleConsecutivePages - CURRENT_PAGE_SLOT_SIZE_1 - pagesBeforeCurrent;

  const startOfMiddleRange = current - pagesBeforeCurrent;
  const endOfMiddleRange = current + pagesAfterCurrent;
  const middlePages = makeInclusiveRange(startOfMiddleRange, endOfMiddleRange);

  return [1, '...', ...middlePages, '...', total];
}

export function makeInclusiveRange(start: number, end: number): Array<number> {
  const result = [];
  for (let i = start; i <= end; i++) result.push(i);
  return result;
}
