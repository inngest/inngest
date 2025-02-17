import { useEffect, useState } from 'react';
import { toast } from 'sonner';

export const devServerURL = 'http://localhost:8288';

function getStreamEventURL(devServerURL: string, eventID: string): string {
  return `${devServerURL}/stream/trigger?event=${eventID}`;
}

// We want the event to appear at the top of the stream, so we omit the timestamp
function omitTimestampFromPayload(payload: string): string {
  try {
    const parsed = JSON.parse(payload);
    delete parsed.ts;
    return JSON.stringify(parsed);
  } catch (err) {
    return payload;
  }
}

async function sendToDevServer(payload: string) {
  try {
    const response = await fetch(`${devServerURL}/e/cloud_ui_key`, {
      body: omitTimestampFromPayload(payload),
      method: 'POST',
      mode: 'cors',
    });
    if (!response.ok) {
      return toast.error('Failed to send to Dev Server');
    }
    const body = await response.json();
    const eventID = body.ids[0];
    toast.success('Sent to Dev Server', {
      description: (
        <>
          Go to:{' '}
          <a href={getStreamEventURL(devServerURL, eventID)} className="underline" target="_blank">
            {eventID}
          </a>
        </>
      ),
    });
  } catch (err) {
    toast.error('Failed to send to Dev Server');
  }
}

export function useDevServer(pollingInterval?: number) {
  const [isRunning, setRunning] = useState<boolean>(false);

  useEffect(() => {
    const checkServerStatus = () => {
      fetch(devServerURL)
        .then(() => setRunning(true))
        .catch(() => setRunning(false));
    };

    checkServerStatus();

    let intervalId: number | undefined;
    if (pollingInterval) {
      intervalId = window.setInterval(checkServerStatus, pollingInterval);
    }

    return () => {
      if (intervalId !== undefined) {
        window.clearInterval(intervalId);
      }
    };
  }, [pollingInterval]);

  return {
    isRunning,
    send: sendToDevServer,
  };
}
