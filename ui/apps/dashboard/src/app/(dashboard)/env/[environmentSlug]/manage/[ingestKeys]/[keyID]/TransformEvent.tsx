'use client';

import { useContext, useEffect, useState } from 'react';
import type { Route } from 'next';
import { usePathname, useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock';
import { toast } from 'sonner';

import { getManageKey } from '@/utils/urls';
import makeVM from '@/utils/vm';
import { Context } from './Context';

type FilterEventsProps = {
  keyID: string;
  keyName: string | null;
  metadata: Record<string, unknown> | null | undefined;
};

const preview = async (transform: string, input: string) => {
  // Add current action metadata
  const vm = await makeVM(Date.now() + 500);
  try {
    // execute expression
    vm.evalCode(transform);
    vm.executePendingJobs(-1);
    const res = vm.evalCode('transform(' + input + ')');
    const unwrapped = vm.unwrapResult(res);
    const ok = vm.dump(unwrapped);
    vm.dispose();
    if (ok) {
      return JSON.stringify(ok, undefined, '  ');
    }
    return ok || '';
  } catch (e) {
    return `Error: ${e}`;
  }
};

const defaultIncoming = `{
  "example": "paste the incoming JSON payload here to test your transform"
}`;

const defaultCommentBlock = `// Rename this webhook to give the events a unique name,
    // or use a field from the incoming event as the event name.`;

// XXX: our server-side JS AST parser does not like ES6 style functions.

export function createTransform({
  eventName = `"webhook/request.received"`,
  dataParam = 'evt',
  commentBlock = defaultCommentBlock,
}): string {
  return `// transform accepts the incoming JSON payload from your
// webhook and must return an object that is in the Inngest event format
function transform(evt, headers = {}, queryParams = {}) {
  return {
    ${commentBlock}
    name: ${eventName},
    data: ${dataParam},
  };
};`;
}
const defaultTransform = createTransform({});

// This must match the output of the default transform and the default incoming!
const defaultOutput = `{
  "name": "webhook/request.received",
  "data": {
    "example": "paste the incoming JSON payload here to test your transform"
  }
}`;

export default function TransformEvents({ keyID, metadata, keyName }: FilterEventsProps) {
  let rawTransform: string | undefined = undefined;
  if (typeof metadata?.transform === 'string') {
    rawTransform = metadata.transform;
  }

  const [transform, setTransform] = useState(rawTransform);
  const [incoming, setIncoming] = useState(defaultIncoming);
  const [isDisabled, setDisabled] = useState(true);
  const [output, setOutput] = useState(defaultOutput);
  const { save } = useContext(Context);
  const router = useRouter();
  const pathname = usePathname();
  const page = getManageKey(pathname);

  const compute = async () => {
    if (!transform) {
      return;
    }

    const result = await preview(transform, incoming);
    setOutput(result);
    if (result === '' || result.indexOf('Error') === 0) {
      return;
    }
  };

  useEffect(() => {
    compute();
  }, [transform, incoming]);

  if (page === 'keys') {
    return null;
  }

  function validateSubmit(nextValue: {}) {
    if (JSON.stringify(nextValue) === JSON.stringify(rawTransform)) {
      setDisabled(true);
    } else {
      setDisabled(false);
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (isDisabled) return;
    save({ id: keyID, metadata: transform ? { transform } : undefined }).then((result) => {
      if (result.error) {
        toast.error(`Webhook could not be updated: ${result.error.message}`);
      } else {
        toast.success('Webhook successfully updated');
        router.refresh();
      }
    });
  }

  function handleTransformCodeChange(code: string) {
    const trimmedCode = code.trim();
    const nextValueEmpty = '';
    const nextValueFull = trimmedCode;
    if (trimmedCode === '') {
      setTransform(nextValueEmpty);
      validateSubmit(nextValueEmpty);
      return;
    }
    setTransform(nextValueFull);
    validateSubmit(nextValueFull);
  }

  function handleIncomingCodeChange(code: string) {
    const trimmedCode = code.trim();
    setIncoming(trimmedCode || defaultIncoming);
  }

  return (
    <form className="pt-3" onSubmit={handleSubmit} id="save-transform">
      <div className="flex">
        <div>
          <h2 className="pb-1 text-lg font-semibold">Transform Event</h2>
          <p className="mb-6 text-sm text-slate-700">
            An optional JavaScript transform used to alter incoming events into our{' '}
            <a
              className="font-semibold text-indigo-500"
              href="https://www.inngest.com/docs/events/event-format-and-structure"
              target="_blank noreferrer"
            >
              event format
            </a>
            .
          </p>
        </div>
        <Button
          href={'https://www.inngest.com/docs/events/event-format-and-structure' as Route}
          appearance="outlined"
          className="ml-auto"
          label="Read Documentation"
        />
      </div>
      <div className="mb-6">
        <CodeBlock
          tabs={[
            {
              label: 'Payload',
              content: rawTransform ?? defaultTransform,
              readOnly: false,
              language: 'javascript',
              handleChange: handleTransformCodeChange,
            },
          ]}
        />
      </div>
      <div className="mb-5 flex gap-5">
        <div className="w-6/12">
          <h2 className="pb-1 text-lg font-semibold">Incoming Event JSON</h2>
          <p className="mb-6 text-sm text-slate-700">
            Paste the incoming JSON payload here to test your transform.
          </p>
          <CodeBlock
            tabs={[
              {
                label: 'Payload',
                content: incoming,
                readOnly: false,
                language: 'json',
                handleChange: handleIncomingCodeChange,
              },
            ]}
          />
        </div>
        <div className="w-6/12">
          <h2 className="pb-1 text-lg font-semibold">Transformed Event</h2>
          <p className="mb-6 text-sm text-slate-700">Preview the transformed JSON payload here.</p>
          <CodeBlock
            tabs={[
              {
                label: 'Payload',
                content: output,
                language: 'json',
              },
            ]}
          />
        </div>
      </div>
      <div className="mb-8 flex justify-end">
        <Button
          kind="primary"
          disabled={isDisabled}
          type="submit"
          label="Save Transform Changes"
          form="save-transform"
        />
      </div>
    </form>
  );
}
