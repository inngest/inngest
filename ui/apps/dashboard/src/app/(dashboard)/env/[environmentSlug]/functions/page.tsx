'use client';

import { useState } from 'react';
import { CodeBracketSquareIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import Header from '@/components/Header/Header';
import LoadingIcon from '@/icons/LoadingIcon';
import FunctionListNotFound from './FunctionListNotFound';
import FunctionStateFilter, { type FunctionState } from './FunctionStateFilter';
import { FunctionTable } from './FunctionTable';
import { useFunctionList } from './useFunctionList';

export const runtime = 'nodejs';

export default function FunctionListPage() {
  // Used to turn non render errors into render errors. This is necessary
  // because React error boundaries only catch errors that occur during
  // rendering.
  const [error, setError] = useState<Error>();
  if (error) {
    throw error;
  }

  const [functionState, setFunctionState] = useState<FunctionState>('active');
  const environment = useEnvironment();

  const [functionList, setFunctionList] = useFunctionList({
    environmentID: environment.id,
    onError: setError,
    functionState,
  });

  function loadMore() {
    setFunctionList((prev) => {
      return {
        ...prev,
        latestRequestedPage: prev.latestRequestedPage + 1,
      };
    });
  }

  const isLoadingInitialData = functionList.latestLoadedPage === 0;
  const environmentHasFunctions = environment.functionCount > 0;

  let content: JSX.Element;
  if (isLoadingInitialData) {
    content = (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  } else if (environmentHasFunctions === false) {
    content = <FunctionListNotFound />;
  } else {
    content = (
      <div className="flex min-h-0 flex-1 flex-col divide-y divide-slate-100">
        <FunctionStateFilter handleClick={setFunctionState} selectedOption={functionState} />
        <FunctionTable rows={functionList.rows} />

        {functionList.hasNextPage && (
          <div className="flex w-full justify-center py-2.5">
            <Button
              disabled={functionList.isLoading}
              appearance="outlined"
              btnAction={loadMore}
              label={functionList.isLoading ? 'Loading' : 'Load More'}
            />
          </div>
        )}
      </div>
    );
  }

  return (
    <>
      <FunctionsHeader />
      {content}
    </>
  );
}

function FunctionsHeader() {
  return (
    <Header title="Functions" icon={<CodeBracketSquareIcon className="h-3.5 w-3.5 text-white" />} />
  );
}
