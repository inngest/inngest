export type CronTriggerMetadata = {
  scheduledAt: Date | null;
  fireAt: Date | null;
};

export function getCronTriggerMetadata(payloads?: string[]): CronTriggerMetadata {
  const payload = payloads?.[0];
  if (!payload) {
    return { scheduledAt: null, fireAt: null };
  }

  try {
    const parsed = JSON.parse(payload);
    const data = parsed?.data;

    return {
      scheduledAt: parseDate(data?.scheduledAt),
      fireAt: parseDate(data?.fireAt),
    };
  } catch {
    return { scheduledAt: null, fireAt: null };
  }
}

function parseDate(value: unknown): Date | null {
  if (typeof value !== 'string' || value === '') {
    return null;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return date;
}
