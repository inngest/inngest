import { format, toDate } from '@inngest/components/utils/date';

export const formatTooltipTimestamp = (value: string) => {
  const date = toDate(value);

  return date ? format(date, 'dd MMM yyyy, HH:mm:ss OOOO') : value;
};
