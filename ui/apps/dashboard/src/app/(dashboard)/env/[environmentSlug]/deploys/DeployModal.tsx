import { useMemo, useState } from 'react';
import { RocketLaunchIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import * as Sentry from '@sentry/nextjs';
import { useLocalStorage } from 'react-use';
import { toast } from 'sonner';

import { Alert } from '@/components/Alert';
import Modal from '@/components/Modal';
import { DOCS_URLS } from '@/utils/urls';
import { DeployFailure } from './DeployFailure';
import DeploySigningKey from './DeploySigningKey';
import { deployViaUrl, type RegistrationErrorCode, type RegistrationFailure } from './utils';

type DeployModalProps = {
  environmentSlug: string;
  isOpen: boolean;
  onClose: () => void;
};

export default function DeployModal({ environmentSlug, isOpen, onClose }: DeployModalProps) {
  const [failure, setFailure] = useState<RegistrationFailure>();
  const [input = '', setInput] = useLocalStorage('deploymentUrl', '');
  const [isLoading, setIsLoading] = useState(false);

  async function onClickDeploy() {
    setIsLoading(true);

    try {
      const failure = await deployViaUrl(input);
      setFailure(failure);
      if (!failure) {
        toast.success('Your app has been deployed!');
        onClose();
      }
    } catch {
      setFailure({
        errorCode: undefined,
        headers: {},
        statusCode: undefined,
      });
    } finally {
      setIsLoading(false);
    }
  }

  /**
   * Disable the button if the URL isn't valid
   */
  const disabled = useMemo(() => {
    try {
      new URL(input);
      return false;
    } catch {
      return true;
    }
  }, [input]);

  return (
    <Modal className="flex max-w-xl flex-col gap-4" isOpen={isOpen} onClose={onClose}>
      <header className="flex flex-row items-center gap-3">
        <RocketLaunchIcon className="h-5 text-indigo-500" />
        <h2 className="text-lg font-medium">Deploy your application</h2>
      </header>
      <p>
        After you&apos;ve set up the{' '}
        <a href={`${DOCS_URLS.SERVE}?ref=app-deploy-modal`} target="_blank noreferrer">
          serve
        </a>{' '}
        API and deployed your application, enter the URL of your application&apos;s serve endpoint
        to register your functions with Inngest.
      </p>
      {/* TODO - Add CTA/info block about Vercel/Netlify integrations */}
      <DeploySigningKey environmentSlug={environmentSlug} />
      <div>
        <input
          className="w-full rounded-lg border px-4 py-2"
          type="text"
          placeholder="https://example.com/api/inngest"
          name="url"
          value={input}
          onChange={(e) => setInput(e.target.value)}
        />
      </div>
      {failure && !isLoading ? <DeployFailure {...failure} /> : null}
      <div className="mt-2 flex flex-row justify-end">
        <Button
          kind="primary"
          className="px-16"
          btnAction={onClickDeploy}
          disabled={disabled || isLoading}
          label="Deploy"
        />
      </div>
    </Modal>
  );
}

type FailureProps = {
  failure: RegistrationFailure;
};

function Failure({ failure }: FailureProps) {
  return (
    <div>
      {createMessage(failure.errorCode)}
      {createDetail(failure)}
    </div>
  );
}

function createDetail(failure: RegistrationFailure) {
  if (
    !failure.errorCode ||
    ![
      'forbidden',
      'internal_server_error',
      'unauthorized',
      'unreachable',
      'url_not_found',
    ].includes(failure.errorCode)
  ) {
    return null;
  }

  let headers: JSX.Element | undefined;
  if (Object.keys(failure.headers).length > 0) {
    headers = (
      <>
        <div>Headers:</div>
        {Object.entries(failure.headers).map(([name, values]) => {
          return (
            <div className="pl-4" key={name}>
              {name}: {values.map((value) => `"${value}"`).join(', ')}
            </div>
          );
        })}
      </>
    );
  }

  let statusCode: JSX.Element | undefined;
  if (failure.statusCode) {
    statusCode = <div>Status code: {failure.statusCode}</div>;
  }

  if (!headers && !statusCode) {
    return;
  }

  return (
    <div>
      <div>This is from your {"app's"} response:</div>
      <pre className="w-full overflow-scroll rounded-md border border-slate-300 bg-slate-100 p-1">
        {statusCode}
        {headers}
      </pre>
    </div>
  );
}

function createMessage(errorCode: RegistrationErrorCode | undefined) {
  let docsURL: string | undefined = undefined;
  let message: string;

  switch (errorCode) {
    case 'forbidden':
      message = 'The request was forbidden. Is the URL behind required authentication?';
      break;
    case 'internal_server_error':
      message = 'Your app had an internal server error.';
      break;
    case 'invalid_function':
      docsURL = 'https://www.inngest.com/docs/reference/functions/create';
      message = 'There is at least 1 invalid function in your app.';
      break;
    case 'invalid_signing_key':
      docsURL = 'https://www.inngest.com/docs/sdk/serve#signing-key';
      message = 'Your app is using an invalid signing key.';
      break;
    case 'missing_signing_key':
      docsURL = 'https://www.inngest.com/docs/sdk/serve#signing-key';
      message = 'Your app does not have a signing key.';
      break;
    case 'no_functions':
      docsURL = 'https://www.inngest.com/docs/reference/serve';
      message = 'Your app does not have any functions.';
      break;
    case 'too_many_pings':
      message =
        "We've received too many requests to register in a short time window. Please slow down your requests.";
      break;
    case 'unauthorized':
      message = 'The request was unauthorized. Please disable authorization on this endpoint.';
      break;
    case 'unreachable':
      message = 'The URL is unreachable. Is the host correct?';
      break;
    case 'unsupported_protocol':
      message = 'Your app does not support this protocol.';
      break;
    case 'url_not_found':
      docsURL = 'https://www.inngest.com/docs/reference/serve';
      message = 'A connection was made to the host but the URL was not found. Is the path correct?';
      break;
    // API didn't provide an error code. We need to change the API to provide an
    // error code, but in the meantime we'll fail gracefully.
    case undefined:
      message = 'Something went wrong.';
      break;
    // There's an error code we aren't handling! We need to add it, but in the
    // meantime we'll fail gracefully.
    default:
      Sentry.captureMessage(`unhandled deploy error code ${errorCode}`, 'error');
      message = 'Something went wrong.';
      break;
  }

  let docsMessage: JSX.Element | undefined;
  if (docsURL) {
    docsMessage = (
      <>
        More information can be found in the docs{' '}
        <a className="text-indigo-500" href={docsURL} target="_blank">
          here
        </a>
        .
      </>
    );
  }

  return (
    <Alert className="mb-4" severity="error">
      {message} {docsMessage}
    </Alert>
  );
}
