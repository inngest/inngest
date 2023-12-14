import { useEffect, useState } from 'react';
import { toast } from 'sonner';

export const devServerURL = 'http://localhost:8288';

function getStreamEventURL(devServerURL: string, eventID: string): string {
  return `${devServerURL}/stream/trigger?event=${eventID}`;
}

async function sendToDevServer(payload: string) {
  try {
    const response = await fetch(`${devServerURL}/e/cloud_ui_key`, {
      body: payload,
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

export function useDevServer() {
  const [isRunning, setRunning] = useState<boolean>(false);
  useEffect(() => {
    fetch(devServerURL)
      .then(() => setRunning(true))
      .catch(() => setRunning(false));
  }, []);
  return {
    isRunning,
    send: (payload: string) => sendToDevServer(payload),
  };
}
