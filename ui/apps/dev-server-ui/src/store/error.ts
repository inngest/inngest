import type { SerializedError } from '@reduxjs/toolkit';

export const convertError = (message: string, error: Error | SerializedError): Error => {
  if (error instanceof Error) {
    return error;
  }
  return new Error(message, {
    cause: JSON.stringify(error),
  });
};
