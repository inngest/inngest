import { useState } from 'react';
import { useAppDispatch } from '@/store/hooks';
import { showFunctions, showDocs } from '@/store/global';
import { type App } from '@/store/generated';
// import { useDeleteAppMutation } from '@/store/devApi';
import CodeLine from '@/components/CodeLine';
import AppCardHeader from '@/components/App/AppCardHeader';
import AppCardStep from './AppCardStep';
import classNames from '@/utils/classnames';
import useInputUrlValidation from '@/hooks/useInputURLValidation';
import {
  IconAppStatusCompleted,
  IconAppStatusFailed,
  IconChevron,
  IconSpinner,
  IconArrowTopRightOnSquare,
  IconAppStatusDefault,
} from '@/icons';

type AppWithoutFunctions = Omit<App, 'functions'>;

export default function AppCard({ app }: { app: AppWithoutFunctions }) {
  const [isAppConnecting, setAppConnecting] = useState(false);
  const [inputUrl, setInputUrl, isUrlInvalid] = useInputUrlValidation({
    callback: () => {
      // To do: edit app, show the isAppConnecting state in the meanwhile, remove it once done
    },
    initialInputValue: app.url ?? undefined,
  });
  // const [_deleteApp, deleteAppState] = useDeleteAppMutation();
  const dispatch = useAppDispatch();

  function handleDelete() {
    // _deleteApp({
    //   id: app.id,
    // });
  }

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    setInputUrl(e.target.value);
  }

  return (
    <div>
      <AppCardHeader
        connected={app.connected}
        functionCount={app.functionCount}
        sdkVersion={app.sdkVersion}
      />
      <div className="border border-slate-700/30 rounded-b-md divide-y divide-slate-700/30 bg-slate-800/30">
        {isAppConnecting ? (
          <div className="p-4 pr-6 flex items-center gap-2">
            <IconSpinner className="fill-sky-400 text-slate-800" />
            <p className="text-slate-400 text-lg font-light">Connecting...</p>
          </div>
        ) : (
          <div className="flex items-center justify-between px-6 py-4 ">
            <p className=" text-lg text-white">{app.name}</p>
            {app.autodiscovered && (
              <span className="text-xs leading-3 border rounded-md border-slate-800 box-border py-1.5 px-2 text-slate-300">
                Auto Detected
              </span>
            )}
          </div>
        )}
        <AppCardStep
          lineContent={
            <>
              <div className="flex items-center gap-3 text-base">
                {app.connected ? (
                  <>{<IconAppStatusCompleted />}Connected to server</>
                ) : (
                  <>{<IconAppStatusFailed />}No connection to server</>
                )}
              </div>
              <div className="flex items-center gap-4">
                <p className="text-slate-300 ui-open:hidden">{app.url}</p>
                <IconChevron className="ui-open:-rotate-180 transform-90 text-slate-500" />
              </div>
            </>
          }
          expandedContent={
            <>
              {!app.connected && (
                <p className="pb-4 text-slate-400">
                  The Inngest Dev Server can’t find your application. Ensure
                  your full URL is correct, including the correct port. Inngest
                  automatically scans{' '}
                  <span className="text-white">multiple ports</span> by default.
                </p>
              )}
              {isUrlInvalid && (
                <p className="pb-4 text-slate-400">Please enter a valid URL</p>
              )}
              <form className="flex items-center justify-between pb-4">
                <label
                  htmlFor="editAppUrl"
                  className="text-sm font-semibold text-white"
                >
                  App URL
                  <span className="text-slate-500 text-sm block">
                    The URL of your application
                  </span>
                </label>
                <div className="relative">
                  <input
                    id="editAppUrl"
                    className={classNames(
                      'min-w-[50%] bg-slate-800 rounded-md text-slate-300 py-2 px-4 outline-2 outline-indigo-500 focus:outline readOnly:outline-transparent',
                      isUrlInvalid && ' outline-rose-500',
                      isAppConnecting && 'pr-6'
                    )}
                    value={inputUrl}
                    placeholder="https://example.com/api/inngest"
                    onChange={handleChange}
                    readOnly={app.autodiscovered}
                  />
                  {isAppConnecting && (
                    <IconSpinner className="absolute top-1/3 right-2 fill-sky-400 text-slate-800" />
                  )}
                </div>
              </form>
              <a
                className="text-indigo-400 flex items-center gap-2 cursor-pointer w-fit"
                onClick={() => dispatch(showDocs('/sdk/serve'))}
              >
                Connecting to the Dev Server
                <IconArrowTopRightOnSquare />
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
                    <IconArrowTopRightOnSquare />
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
