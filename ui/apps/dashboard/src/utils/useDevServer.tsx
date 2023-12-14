import { useEffect, useState } from 'react';
import { toast } from 'sonner';

const devServerUrl = 'http://localhost:8288';

async function sendToDevServer(payload: string) {
  try {
    await fetch(`${devServerUrl}/e/cloud_ui_key`, {
      body: payload,
      method: 'POST',
      mode: 'cors',
    });
    toast.success('Sent to Dev Server', {
      description: (
        <>
          View at{' '}
          <a href={devServerUrl} className="underline" target="_blank">
            {devServerUrl.replace(/https?:\/\//, '')}
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
    fetch(devServerUrl)
      .then(() => setRunning(true))
      .catch(() => setRunning(false));
  }, []);
  return {
    isRunning,
    send: (payload: string) => sendToDevServer(payload),
  };
}
