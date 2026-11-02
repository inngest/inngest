import { format, formatInTimeZone } from './date';

export const formatLocalTime = (date: Date) => format(date, 'dd MMM yyyy, hh:mm:ss a');

export const formatUTCTime = (date: Date) => formatInTimeZone(date, 'UTC', 'dd MMM yyyy, HH:mm:ss');
