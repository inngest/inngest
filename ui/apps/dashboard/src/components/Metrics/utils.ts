import { lightFormat, toDate } from '@inngest/components/utils/date';

export const dateFormat = (dateString: string, diff: number) => {
  const date = toDate(dateString);
  if (!date) {
    return dateString;
  }

  const d = Math.abs(diff);

  return d < 6000 // a minute
    ? lightFormat(date, 'HH:mm:ss:SSS')
    : d <= 8.64e7 // a day
    ? lightFormat(date, 'HH:mm')
    : lightFormat(date, 'MM/dd:HH');
};
