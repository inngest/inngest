import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock/CodeBlock';
import { parseCode } from '@inngest/components/InvokeButton/utils';
import { Link } from '@inngest/components/Link';
import { Select, type Option } from '@inngest/components/Select/Select';
import { RiCheckboxCircleFill } from '@remixicon/react';
import { toast } from 'sonner';

import { pathCreator } from '@/utils/urls';
import { OnboardingSteps } from '../Onboarding/types';
import { invokeFunction, prefetchFunctions } from './actions';
import useOnboardingStep from './useOnboardingStep';
import { useOnboardingTracking } from './useOnboardingTracking';

const initialCode = JSON.stringify(
  {
    data: {
      example: 'type a JSON payload here to test your function',
    },
  },
  null,
  2
);

interface FunctionOption extends Option {
  slug: string;
  current: {
    triggers: {
      eventName?: string;
    }[];
  };
}

export default function InvokeFn() {
  const currentStepName = OnboardingSteps.InvokeFn;
  const { updateCompletedSteps, lastCompletedStep } = useOnboardingStep();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [functions, setFunctions] = useState<FunctionOption[]>([]);
  const [selectedFunction, setSelectedFunction] = useState<FunctionOption | null>(null);
  const [rawPayload, setRawPayload] = useState(initialCode);
  const [isFnInvoked, setIsFnInvoked] = useState(false);
  const router = useRouter();
  const tracking = useOnboardingTracking();

  const isOnboardingCompleted = lastCompletedStep?.isFinalStep;

  const hasEventTrigger =
    selectedFunction?.current.triggers.some((trigger) => trigger.eventName) ?? false;

  useEffect(() => {
    const loadFunctions = async () => {
      try {
        setLoading(true);
        const fetchedFunctions = await prefetchFunctions();
        setFunctions(fetchedFunctions);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load functions');
        console.error('Failed to load functions: ', err);
      } finally {
        setLoading(false);
      }
    };

    loadFunctions();
  }, []);

  const handleInvokeFn = async () => {
    if (!selectedFunction || !selectedFunction.slug) return;
    let payload: ReturnType<typeof parseCode>;
    if (hasEventTrigger) {
      payload = parseCode(rawPayload);
    } else {
      payload = { data: {}, user: null };
    }
    try {
      const { success, error } = await invokeFunction({
        functionSlug: selectedFunction.slug,
        user: payload.user,
        data: payload.data,
      });
      if (success) {
        updateCompletedSteps(currentStepName, {
          metadata: {
            completionSource: 'manual',
            invokedFunction: selectedFunction,
          },
        });
        setError(undefined);
        setIsFnInvoked(true);
        // TO DO: add link to run ID, need to update mutation first to return ID
        toast.success('Function successfully invoked');
      } else {
        setIsFnInvoked(false);
        setError(error);
        console.error('Failed to invoke: ', error);
      }
    } catch (err) {
      setError('An error occurred');
      console.error('Failed to invoke: ', err);
    }
  };

  const selectDisplay = loading
    ? 'Loading functions...'
    : functions.length === 0
    ? 'No functions'
    : 'Select function';

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        You can send a test event and see your function in action. You will be able to access all
        our monitoring and debugging features.{' '}
        <Link
          className="inline-block"
          size="small"
          href="https://www.inngest.com/docs/features/events-triggers?ref=app-onboarding-invoke-fn"
          target="_blank"
        >
          Read more
        </Link>
      </p>
      <div className="border-subtle my-6 rounded-md border px-6 py-4">
        <p className="text-muted mb-2 text-sm font-medium">Select function to test:</p>

        <Select
          onChange={(option) => setSelectedFunction(option as FunctionOption)}
          isLabelVisible={false}
          label={selectDisplay}
          multiple={false}
          value={selectedFunction}
          className="mb-6"
        >
          <Select.Button
            className={functions.length === 0 ? 'text-disabled cursor-not-allowed' : ''}
          >
            <div className="text-sm font-medium leading-tight">
              {selectedFunction?.name || selectDisplay}
            </div>
          </Select.Button>
          {functions.length > 0 && (
            <Select.Options className="w-full">
              {functions.map((option) => {
                return (
                  <Select.Option key={option.id} option={option}>
                    {option.name}
                  </Select.Option>
                );
              })}
            </Select.Options>
          )}
        </Select>
        {functions.length === 0 && !loading && (
          <Alert className="mt-6" severity="warning">
            <p className="text-sm">Make sure your app is synced and has functions.</p>
          </Alert>
        )}
        {hasEventTrigger && (
          <CodeBlock.Wrapper>
            <CodeBlock
              header={{ title: 'Invoke function' }}
              tab={{
                content: rawPayload,
                readOnly: false,
                language: 'json',
                handleChange: setRawPayload,
              }}
            />
          </CodeBlock.Wrapper>
        )}
        {!hasEventTrigger && selectedFunction && (
          <p className="text-sm">
            Cron functions without event triggers cannot include payload data.
          </p>
        )}

        {error && (
          <Alert className="mt-6" severity="error">
            <p className="text-sm">{error}</p>
          </Alert>
        )}
        <div className="mt-6 flex items-center justify-between">
          {!isFnInvoked ? (
            <div className="flex items-center gap-2">
              <Button
                label="Invoke test function"
                disabled={!selectedFunction}
                onClick={() => {
                  tracking?.trackOnboardingAction(currentStepName, {
                    metadata: {
                      type: 'btn-click',
                      label: 'invoke',
                      invokedFunction: selectedFunction,
                    },
                  });
                  handleInvokeFn();
                }}
              />
              <Button
                appearance="outlined"
                label="Skip, take me to dashboard"
                onClick={() => {
                  updateCompletedSteps(currentStepName, {
                    metadata: {
                      completionSource: 'manual',
                      invokedFunction: null,
                    },
                  });
                  tracking?.trackOnboardingAction(currentStepName, {
                    metadata: {
                      type: 'btn-click',
                      label: 'skip',
                      invokedFunction: selectedFunction,
                    },
                  });
                  router.push(pathCreator.apps({ envSlug: 'production' }));
                }}
              />
            </div>
          ) : (
            <Button
              label="Go to runs"
              onClick={() => {
                tracking?.trackOnboardingAction(currentStepName, {
                  metadata: {
                    type: 'btn-click',
                    label: 'go-to-runs',
                    invokedFunction: selectedFunction,
                  },
                });
                router.push(pathCreator.runs({ envSlug: 'production' }));
              }}
            />
          )}

          {isOnboardingCompleted && (
            <div className="text-success flex items-center gap-0.5 text-sm">
              <RiCheckboxCircleFill className="h-4 w-4" />
              Onboarding completed
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
