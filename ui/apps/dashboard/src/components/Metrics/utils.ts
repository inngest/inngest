import { lightFormat } from '@inngest/components/utils/date';

export const dateFormat = (dateString: string, diff: number) => {
  const d = Math.abs(diff);

  return d < 6000 // a minute
    ? lightFormat(dateString, 'HH:mm:ss:SSS')
    : d <= 8.64e7 // a day
    ? lightFormat(dateString, 'HH:mm')
    : lightFormat(dateString, 'MM/dd:HH');
};
