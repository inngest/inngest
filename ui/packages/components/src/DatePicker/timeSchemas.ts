import { z } from 'zod';

type months = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

const MIN_HOUR_AM = 1;
const MAX_HOUR_AM = 12;
const MIN_HOUR_24 = 0;
const MAX_HOUR_24 = 23;

function getMinMaxHour(is24Format: boolean) {
  if (is24Format) {
    return { minHour: MIN_HOUR_24, maxHour: MAX_HOUR_24 };
  } else {
    return { minHour: MIN_HOUR_AM, maxHour: MAX_HOUR_AM };
  }
}

export const getHourSchema = (is24Format: boolean) => {
  const { minHour, maxHour } = getMinMaxHour(is24Format);

  return z.coerce
    .number()
    .int()
    .min(minHour, { message: `Hours cannot be less than ${minHour}` })
    .max(maxHour, { message: `Hours cannot exceed ${maxHour}` });
};

export const minutesSchema = z.coerce
  .number()
  .int()
  .min(0, { message: 'Minutes cannot be less than 0' })
  .max(59, { message: 'Minutes cannot exceed 59' });

export const secondsSchema = z.coerce
  .number()
  .int()
  .min(0, { message: 'Seconds cannot be less than 0' })
  .max(59, { message: 'Seconds cannot exceed 59' });

export const millisecondsSchema = z.coerce
  .number()
  .int()
  .min(0, { message: 'Milliseconds cannot be less than 0' })
  .max(999, { message: 'Milliseconds cannot exceed 999' });

export const periodSchema = z
  .string()
  .transform((val) => val.toLowerCase())
  .refine((value) => value === 'am' || value === 'pm', {
    message: 'Period must be "AM" or "PM".',
  });

export const yearSchema = z.coerce
  .number()
  .int()
  .min(1970, { message: 'Year must be a 4 digit number' })
  .max(3000, { message: 'Year must be a 4 digit number' });

export const monthSchema = z.coerce
  .number()
  .int()
  .min(1, { message: 'Month must be between 1 and 12' })
  .max(12, { message: 'Month must be 1 - 12' });
