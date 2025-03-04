export function useTracking() {
  const trackEvent = async (eventName: string, eventData: Record<string, any> = {}) => {
    console.log(`Tracking event: ${eventName}`, eventData);
    // try {
    //   const response = await fetch('/api/telemetry', {
    //     method: 'POST',
    //     headers: {
    //       'Content-Type': 'application/json',
    //     },
    //     body: JSON.stringify({
    //       eventName,
    //       data: eventData,
    //     }),
    //   });

    //   if (!response.ok) {
    //     console.error('Failed to send telemetry event');
    //   }
    // } catch (err) {
    //   console.error(err instanceof Error ? err : new Error('An error occurred'));
    // }
  };

  return { trackEvent };
}
