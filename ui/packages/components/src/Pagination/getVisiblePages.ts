type GetVisiblePagesConfig = {
  current: number;
  total: number;
  variant?: 'normal' | 'narrow';
};

type PaginationConfig = {
  current: number;
  total: number;
  beginningConsecutivePages: number;
  endConsecutivePages: number;
  middleConsecutivePages: number;
  beginningThreshold: number;
  endThreshold: number;
};

export function getVisiblePages({
  current,
  total,
  variant = 'normal',
}: GetVisiblePagesConfig): Array<number | '...'> {
  const slots = variant === 'narrow' ? 5 : 7;

  // How many consecutive pages each pattern shows (calculated to fill exactly slots)
  const beginningConsecutivePages = slots - 1 - 1; // slots - ellipsis - last_page
  const endConsecutivePages = slots - 1 - 1; // slots - first_page - ellipsis
  const middleConsecutivePages = slots - 1 - 2 - 1; // slots - first_page - 2_ellipses - last_page

  if (slots >= total) return makeInclusiveRange(1, total);

  // Transition early on both ends to keep current page position stable within slot array.
  const beginningThreshold = beginningConsecutivePages - 1;
  const endThreshold = total - (endConsecutivePages - 1) + 1;

  const config: PaginationConfig = {
    current,
    total,
    beginningConsecutivePages,
    endConsecutivePages,
    middleConsecutivePages,
    beginningThreshold,
    endThreshold,
  };

  if (current <= beginningThreshold) return handleBeginningCase(config);
  else if (current >= endThreshold) return handleEndCase(config);
  else return handleMiddleCase(config);
}

// Pattern: [first beginningConsecutivePages, '...', total]
function handleBeginningCase({
  beginningConsecutivePages,
  total,
}: PaginationConfig): Array<number | '...'> {
  return [...makeInclusiveRange(1, beginningConsecutivePages), '...', total];
}

// Pattern: [1, '...', last endConsecutivePages]
function handleEndCase({ endConsecutivePages, total }: PaginationConfig): Array<number | '...'> {
  const startOfConsecutive = total - (endConsecutivePages - 1);
  return [1, '...', ...makeInclusiveRange(startOfConsecutive, total)];
}

// Pattern: [1, '...', current-X, current, current+X, '...', total]
function handleMiddleCase({
  current,
  middleConsecutivePages,
  total,
}: PaginationConfig): Array<number | '...'> {
  const pagesBeforeCurrent = Math.floor((middleConsecutivePages - 1) / 2);
  const pagesAfterCurrent = middleConsecutivePages - 1 - pagesBeforeCurrent;

  const startOfMid = current - pagesBeforeCurrent;
  const endOfMid = current + pagesAfterCurrent;

  return [1, '...', ...makeInclusiveRange(startOfMid, endOfMid), '...', total];
}

function makeInclusiveRange(start: number, end: number): Array<number> {
  const result = [];
  for (let i = start; i <= end; i++) result.push(i);
  return result;
}
