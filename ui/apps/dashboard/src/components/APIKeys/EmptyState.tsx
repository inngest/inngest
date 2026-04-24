import { Button } from '@inngest/components/Button';
import { RiExchange2Line } from '@remixicon/react';

import { CreateAPIKeyButton } from './CreateAPIKeyButton';

type Props = {
  onCreate: () => void;
};

export function APIKeysEmptyState({ onCreate }: Props) {
  return (
    <div className="border-muted flex flex-col items-center gap-5 rounded-lg border px-6 py-12">
      <div className="bg-canvasSubtle flex h-14 w-14 items-center justify-center rounded-lg">
        <RiExchange2Line className="text-subtle h-7 w-7" />
      </div>
      <div className="flex flex-col items-center gap-1.5 text-center">
        <p className="text-basis text-xl">Create API key</p>
        <p className="text-subtle max-w-[617px] text-sm">
          API keys are shared credentials that let your applications securely
          connect to Inngest. Generate a key to start running functions and
          managing workflows.
        </p>
      </div>
      <div className="flex gap-3">
        <CreateAPIKeyButton onClick={onCreate} />
        <Button
          kind="primary"
          appearance="outlined"
          label="Go to docs"
          href="https://www.inngest.com/docs"
        />
      </div>
    </div>
  );
}
