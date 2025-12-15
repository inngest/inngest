import { toast } from 'sonner';

export interface EventPayload {
  name: string;
  data?: Record<string, unknown>;
  user?: Record<string, unknown>;
  ts?: number;
  id?: string;
}

export const generateSDKCode = (payload: EventPayload | EventPayload[]): string => {
  return `import { Inngest } from 'inngest';

const inngest = new Inngest({ name: 'My App' });

await inngest.send(${JSON.stringify(payload, null, 2)});`;
};

export const generateCurlCode = (payload: EventPayload | EventPayload[]): string => {
  return `curl http://localhost:8288/e/dev_key \\
  -H "Content-Type: application/json" \\
  --data '${JSON.stringify(payload)}'`;
};

export const copyToClipboard = async (text: string): Promise<void> => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard!');
  } catch (error) {
    toast.error('Failed to copy to clipboard');
  }
};
