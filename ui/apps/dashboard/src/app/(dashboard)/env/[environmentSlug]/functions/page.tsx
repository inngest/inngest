'use client';

import { useState } from 'react';
import { CodeBracketSquareIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import Header from '@/components/Header/Header';
import LoadingIcon from '@/icons/LoadingIcon';
import { useEnvironment } from '@/queries';
import FunctionListNotFound from './FunctionListNotFound';
import FunctionStateFilter, { type FunctionState } from './FunctionStateFilter';
import { FunctionTable } from './FunctionTable';
import { useFunctionList } from './useFunctionList';

export const runtime = 'nodejs';

type FunctionListPageProps = {
  params: {
    environmentSlug: string;
  };
};

export default function FunctionListPage({ params }: FunctionListPageProps) {
  // Used to turn non render errors into render errors. This is necessary
  // because React error boundaries only catch errors that occur during
  // rendering.
  const [error, setError] = useState<Error>();
  if (error) {
    throw error;
  }

  const [functionState, setFunctionState] = useState<FunctionState>('active');

  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug: params.environmentSlug,
  });
  if (!isFetchingEnvironment && !environment) {
    throw new Error('unable to load environment');
  }

  const [functionList, setFunctionList] = useFunctionList({
    environmentID: environment?.id,
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

  const isLoadingInitialData = isFetchingEnvironment || functionList.latestLoadedPage === 0;
  const environmentHasFunctions = environment ? environment.functionCount > 0 : undefined;

  let content: JSX.Element;
  if (isLoadingInitialData) {
    content = (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  } else if (environmentHasFunctions === false) {
    content = <FunctionListNotFound environmentSlug={params.environmentSlug} />;
  } else {
    content = (
      <div className="flex min-h-0 flex-1 flex-col divide-y divide-slate-100">
        <FunctionStateFilter handleClick={setFunctionState} selectedOption={functionState} />
        <FunctionTable environmentSlug={params.environmentSlug} rows={functionList.rows} />

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
