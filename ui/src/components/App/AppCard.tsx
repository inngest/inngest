import { useState } from 'react';
import { toast } from 'sonner';
import { useAppDispatch } from '@/store/hooks';
import { showFunctions, showDocs } from '@/store/global';
import { type App } from '@/store/generated';
import CodeLine from '@/components/CodeLine';
import AppCardHeader from '@/components/App/AppCardHeader';
import Badge from '@/components/Badge';
import AppCardStep from './AppCardStep';
import classNames from '@/utils/classnames';
import { useUpdateAppMutation, useDeleteAppMutation } from '@/store/generated';
import useDebounce from '@/hooks/useDebounce';
import isValidUrl from '@/utils/urlValidation';
import {
  IconAppStatusCompleted,
  IconAppStatusFailed,
  IconChevron,
  IconSpinner,
  IconBook,
  IconAppStatusDefault,
} from '@/icons';

type AppWithoutFunctions = Omit<App, 'functions'>;

export default function AppCard({ app }: { app: AppWithoutFunctions }) {
  const [inputUrl, setInputUrl] = useState(app.url || '');
  const [isUrlInvalid, setUrlInvalid] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const dispatch = useAppDispatch();
  const [_updateApp, updateAppState] = useUpdateAppMutation();
  const [_deleteApp, deleteAppState] = useDeleteAppMutation();

  const debouncedRequest = useDebounce(() => {
    if (isValidUrl(inputUrl)) {
      setUrlInvalid(false);
      updateApp();
    } else {
      setUrlInvalid(true);
      setIsLoading(false);
    }
  });

  async function updateApp() {
    try {
      const response = await _updateApp({
        input: {
          url: inputUrl,
          id: app.id,
        },
      });
      toast.success('The URL was successfully updated.');
      console.log('Edited app URL:', response);
    } catch (error) {
      toast.error('The URL could not be updated.');
      console.error('Error editing app:', error);
    }
    setIsLoading(false);
  }

  async function deleteApp() {
    try {
      const response = await _deleteApp({
        id: app.id,
      });
      toast.success(`${app.name || 'The app'} was successfully deleted.`);
      console.log('Deleted app:', response);
    } catch (error) {
      toast.error(`${app.name || 'The app'} could not be deleted: ${error}`);
      console.error('Error deleting app:', error);
    }
    // To do: add optimistic render in the list
  }

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    setInputUrl(e.target.value);
    setIsLoading(true);
    debouncedRequest();
  }

  function handleDelete() {
    deleteApp();
  }

  return (
    <div>
      <AppCardHeader
        connected={app.connected}
        functionCount={app.functionCount}
        sdkVersion={app.sdkVersion}
      />
      <div className="border border-slate-700/30 rounded-b-md divide-y divide-slate-700/30 bg-slate-800/30">
        {!app.name ? (
          <div className="p-4 pr-6 flex items-center gap-2">
            <IconSpinner className="fill-sky-400 text-slate-800" />
            <p className="text-slate-400 text-lg font-normal">Connecting...</p>
          </div>
        ) : (
          <div className="flex items-center justify-between px-6 py-4 ">
            {!app.connected ? (
              <div className="flex items-center gap-2">
                <IconSpinner className="fill-sky-400 text-slate-800" />
                <p className="text-slate-400 text-lg font-normal">
                  Connecting to {app.name}...
                </p>
              </div>
            ) : (
              <p className=" text-lg text-white">{app.name}</p>
            )}
            {app.autodiscovered && <Badge>Auto Detected</Badge>}
          </div>
        )}
        <AppCardStep
          lineContent={
            <>
              <div className="">
                <div className='flex items-center gap-3 text-base'>
                {app.connected ? (
                  <>{<IconAppStatusCompleted />}Connected to server</>
                ) : (
                  <>{<IconAppStatusFailed />}No connection to server</>
                )}
                </div>
                <p className="text-slate-300 ui-open:hidden xl:hidden pl-10">{app.url}</p>
              </div>
              <div className="flex items-center gap-4">
                <p className="text-slate-300 xl:flex xl:ui-open:hidden hidden">{app.url}</p>
                <IconChevron className="ui-open:-rotate-180 transform-90 text-slate-500" />
              </div>
            </>
          }
          expandedContent={
            <>
              {!app.connected && (
                <>
                  <p className="pb-4 text-slate-400">
                    The Inngest Dev Server can’t find your application. Ensure
                    your full URL is correct, including the correct port.
                    Inngest automatically scans{' '}
                    <span className="text-white">multiple ports</span> by
                    default.
                  </p>
                  {app.error && (
                    <p className="pb-4 text-rose-400 font-medium	">
                      Error: {app.error}
                    </p>
                  )}
                </>
              )}
              <form className="block xl:flex xl:items-center xl:justify-between pb-4">
                <label
                  htmlFor="editAppUrl"
                  className="text-sm font-semibold text-white"
                >
                  App URL
                  <span className="text-slate-500 text-sm block">
                    The URL of your application
                  </span>
                </label>
                <div className="relative flex-1 pt-2 xl:pl-10 xl:pt-0">
                  <input
                    id="editAppUrl"
                    className={classNames(
                      'w-full bg-slate-800 rounded-md text-slate-300 py-2 px-4 outline-2 outline-indigo-500 focus:outline read-only:outline-transparent',
                      isUrlInvalid && ' outline-rose-400',
                      isLoading && 'pr-6'
                    )}
                    value={inputUrl}
                    placeholder="http://localhost:3000/api/inngest"
                    onChange={handleChange}
                    readOnly={app.autodiscovered}
                  />
                  {isLoading && (
                    <IconSpinner className="absolute top-1/3 right-2 fill-sky-400 text-slate-800" />
                  )}
                  {isUrlInvalid && (
                    <p className="absolute text-rose-400 top-10 left-14">
                      Please enter a valid URL
                    </p>
                  )}
                </div>
              </form>
              <a
                className="text-indigo-400 flex items-center gap-2 cursor-pointer w-fit"
                onClick={() => dispatch(showDocs('/sdk/serve'))}
              >
                Connecting to the Dev Server
                <IconBook />
              </a>
            </>
          }
        />
        <AppCardStep
          isEvenStep
          isExpandable={!app.connected || app.functionCount < 1}
          lineContent={
            <>
              <div className="flex items-center gap-3 text-base">
                {app.connected && app.functionCount > 0 ? (
                  <>
                    {<IconAppStatusCompleted />}
                    {app.functionCount} Functions registered
                  </>
                ) : !app.connected ? (
                  <>{<IconAppStatusDefault />}No Functions Found</>
                ) : (
                  <>{<IconAppStatusFailed />}No Functions Found</>
                )}
              </div>
              <div className="flex items-center gap-4">
                {app.connected && app.functionCount > 0 ? (
                  <>
                    <button
                      className="text-indigo-400 flex items-center gap-2"
                      onClick={() => dispatch(showFunctions())}
                    >
                      View Functions
                      <IconChevron className="-rotate-90" />
                    </button>
                  </>
                ) : (
                  <IconChevron className="ui-open:-rotate-180 transform-90 text-slate-500" />
                )}
              </div>
            </>
          }
          expandedContent={
            <>
              {(!app.connected || app.functionCount < 1) && (
                <>
                  <p className="pb-4 text-slate-400">
                    There are currently no functions registered at this url.
                    Ensure you have created a function and are exporting it
                    correctly from your serve command.
                  </p>
                  <CodeLine
                    code="serve(client, [list_of_fns]);"
                    className="p-4 mb-4"
                  />
                  <a
                    className="text-indigo-400 flex items-center gap-2 cursor-pointer w-fit"
                    onClick={() => dispatch(showDocs('/functions'))}
                  >
                    Creating Functions
                    <IconBook />
                  </a>
                </>
              )}
            </>
          }
        />
        {!app.autodiscovered && (
          <div className="text-white p-4 pr-6">
            <button className="text-rose-400" onClick={handleDelete}>
              Delete App
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
