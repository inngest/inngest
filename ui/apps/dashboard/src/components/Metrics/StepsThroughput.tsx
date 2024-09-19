import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';

export const StepsThroughput = () => {
  return (
    <div className="bg-canvasBase border-subtle overflowx-hidden relative flex h-[300px] w-full flex-col rounded-lg p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Total steps throughput{' '}
          <Info
            text="Total number of steps processed your env, app or function."
            action={
              <NewLink
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/features/inngest-functions/steps-workflows"
              >
                Learn more about step throughput.
              </NewLink>
            }
          />
        </div>
      </div>
      <div className="flex h-full flex-row items-center">
        {/* <Chart option={{}} className="h-full w-full" /> */}
        coming soon...
      </div>
    </div>
  );
};
