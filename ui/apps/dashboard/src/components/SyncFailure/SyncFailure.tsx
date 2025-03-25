import { Alert } from '@inngest/components/Alert';
import { cn } from '@inngest/components/utils/classNames';

import { httpDataSchema, type CodedError } from '@/codedError';
import { getMessage } from './utils';

type Props = {
  error: CodedError;
  className?: string;
};

export function SyncFailure({ error, className }: Props) {
  let dataJSON: unknown;
  if (typeof error.data === 'string') {
    try {
      dataJSON = JSON.parse(error.data);
    } catch (e) {
      // noop
    }
  }

  const httpData = httpDataSchema.safeParse(dataJSON);

  return (
    <Alert className={cn('my-2', className)} severity="error">
      <span>{getMessage(error)}</span>

      {httpData.success && httpData.data.statusCode > 0 && (
        <div className="mt-2">Status code: {httpData.data.statusCode}</div>
      )}

      {httpData.success && Object.keys(httpData.data.headers).length > 0 && (
        <div className="mt-2">
          Headers:
          <pre>
            <code className="pl-4">
              {Object.entries(httpData.data.headers).map(([k, v]) => {
                return `\t${k}: ${v.join(', ')}\n`;
              })}
            </code>
          </pre>
        </div>
      )}
    </Alert>
  );
}
