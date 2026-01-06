import { Button } from '@inngest/components/Button';
import { Error } from '@inngest/components/Error/Error';
import type { ErrorComponentProps } from '@tanstack/react-router';
import { rootRouteId, useMatch, useRouter } from '@tanstack/react-router';

import * as Sentry from '@sentry/tanstackstart-react';

function DefaultCatchBoundary({ error }: ErrorComponentProps) {
  const router = useRouter();
  const isRoot = useMatch({
    strict: false,
    select: (state) => state.id === rootRouteId,
  });

  console.error(error.message);

  return (
    <div className="flex flex-col justify-start items-start gap">
      <Error message={error.message} />

      <div className="flex gap-2 justif-start items-center flex-wrap mx-4">
        <Button
          kind="secondary"
          appearance="outlined"
          onClick={() => {
            router.invalidate();
          }}
          label="Try Again"
        />

        {isRoot ? (
          <Button to="/" kind="secondary" appearance="outlined" label="Home" />
        ) : (
          <Button
            kind="secondary"
            appearance="outlined"
            onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
              e.preventDefault();
              window.history.back();
            }}
            label="Go Back"
          />
        )}
      </div>
    </div>
  );
}

export const SentryWrappedCatchBoundary = Sentry.withErrorBoundary(
  DefaultCatchBoundary,
  {},
);
