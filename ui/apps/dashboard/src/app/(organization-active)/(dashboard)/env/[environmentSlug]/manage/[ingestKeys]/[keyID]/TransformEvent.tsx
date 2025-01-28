'use client';

import { useContext, useEffect, useState } from 'react';
import type { Route } from 'next';
import { usePathname, useRouter } from 'next/navigation';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { toast } from 'sonner';

import DashboardCodeBlock from '@/components/DashboardCodeBlock/DashboardCodeBlock';
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
    const escapedInput = JSON.stringify(input);
    const res = vm.evalCode(`transform(${input}, {}, {}, ${escapedInput})`);
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
// webhook and must return an object that is in the Inngest event format.
//
// The raw argument is the original stringified request body. This is useful
// when you want to perform HMAC validation within your Inngest functions.
function transform(evt, headers = {}, queryParams = {}, raw = "") {
  return {
    ${commentBlock}
    name: ${eventName},
    data: ${dataParam},
  };
};`;
}
export const defaultTransform = createTransform({});

// This must match the output of the default transform and the default incoming!
const defaultOutput = `{
  "name": "webhook/request.received",
  "data": {
    "example": "paste the incoming JSON payload here to test your transform"
  }
}`;

export default function TransformEvents({ keyID, metadata }: FilterEventsProps) {
  let rawTransform: string | undefined = undefined;
  if (typeof metadata?.transform === 'string') {
    rawTransform = metadata.transform;
  }

  const [transform, setTransform] = useState(rawTransform);
  const [incoming, setIncoming] = useState(defaultIncoming);
  const [isDisabled, setDisabled] = useState(true);
  const [output, setOutput] = useState(defaultOutput);
  const [outputError, setOutputError] = useState<string | null>(null);
  const [transformWarningOnKey, setTransformWarningOnKey] = useState<string | null>(null);
  const { save } = useContext(Context);
  const router = useRouter();
  const pathname = usePathname();
  const page = getManageKey(pathname);

  const compute = async () => {
    if (!transform) {
      return;
    }

    const result = await preview(transform, incoming);
    if (result.startsWith('Error: ')) {
      setTransformWarningOnKey(null);
      setOutputError(result.replace('Error: ', ''));
      setOutput(defaultOutput);
    } else {
      setTransformWarningOnKey(null);
      setOutputError(null);
      setOutput(result);

      try {
        const parsed = JSON.parse(result);
        if (parsed['name'] === undefined) {
          setTransformWarningOnKey('name');
        } else if (parsed['data'] === undefined) {
          setTransformWarningOnKey('data');
        }
      } catch (e) {
        setTransformWarningOnKey('The resulting output is not a valid JSON object.');
      }
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
      <div className="flex justify-between">
        <div>
          <h2 className="pb-1 text-lg font-semibold">Transform Event</h2>
          <p className="text-subtle mb-6 text-sm">
            An optional JavaScript transform used to alter incoming events into our{' '}
            <Link
              className="inline-flex"
              href="https://www.inngest.com/docs/events/event-format-and-structure"
              target="_blank"
            >
              event format
            </Link>
            .
          </p>
        </div>
        <Button
          href={'https://www.inngest.com/docs/events/event-format-and-structure' as Route}
          appearance="outlined"
          kind="secondary"
          className="ml-auto"
          label="Read documentation"
        />
      </div>
      <div className="mb-6">
        <DashboardCodeBlock
          header={{
            title: 'Transform Function',
          }}
          tab={{
            content: rawTransform ?? defaultTransform,
            readOnly: false,
            language: 'javascript',
            handleChange: handleTransformCodeChange,
          }}
        />
      </div>
      {outputError && (
        <Alert severity="error" className="mb-4">
          <span className="font-bold">JavaScript Error:</span> {outputError}
        </Alert>
      )}
      <div className="mb-5 flex gap-5">
        <div className="w-6/12">
          <h2 className="pb-1 text-lg font-semibold">Incoming Event JSON</h2>
          <p className="text-subtle mb-6 text-sm">
            Paste the incoming JSON payload here to test your transform.
          </p>
          <DashboardCodeBlock
            header={{
              title: 'Webhook Payload',
            }}
            tab={{
              content: incoming,
              readOnly: false,
              language: 'json',
              handleChange: handleIncomingCodeChange,
            }}
          />
        </div>
        <div className="w-6/12">
          <h2 className="pb-1 text-lg font-semibold">Transformed Event</h2>
          <p className="text-subtle mb-6 text-sm">Preview the transformed JSON payload here.</p>
          <DashboardCodeBlock
            header={{
              title: 'Event Payload',
            }}
            tab={{
              content: output,
              language: 'json',
            }}
          />
          {transformWarningOnKey && (
            <Alert severity="warning" className="mt-4">
              The resulting output is missing a <code>{transformWarningOnKey}</code> field and is
              not{' '}
              <a
                href="https://www.inngest.com/docs/features/events-triggers/event-format"
                className={'underline'}
                target={'_blank'}
              >
                a valid Inngest event
              </a>
              .
            </Alert>
          )}
        </div>
      </div>
      <div className="mb-8 flex justify-end">
        <Button
          kind="primary"
          disabled={isDisabled}
          type="submit"
          label="Save transform changes"
          form="save-transform"
        />
      </div>
    </form>
  );
}
