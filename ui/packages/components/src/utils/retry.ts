async function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Retry a callback until it succeeds or the maximum number of attempts is
 * reached
 */
export async function withRetry<T>(callback: () => Promise<T>): Promise<T> {
  const maxAttempts = 5;
  let attempt = 0;
  while (true) {
    attempt += 1;

    try {
      return await callback();
    } catch (error) {
      if (attempt >= maxAttempts) {
        throw error;
      }
      await sleep(1_000 * attempt);
    }
  }
}
