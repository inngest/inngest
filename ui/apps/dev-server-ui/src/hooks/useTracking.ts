export function useTracking() {
  const trackEvent = async (eventName: string, eventData: Record<string, any> = {}) => {
    try {
      const response = await fetch(createDevServerURL('/v0/telemetry'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          eventName,
          ...eventData,
        }),
      });

      if (!response.ok) {
        console.error('Failed to send telemetry event');
      }
    } catch (err) {
      console.error(err instanceof Error ? err : new Error('An error occurred'));
    }
  };

  return { trackEvent };
}

function createDevServerURL(path: string) {
  const host = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!host) {
    return path;
  }
  return new URL(path, host).toString();
}
