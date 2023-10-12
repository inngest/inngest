import { Alert } from '@/components/Alert';
import type { RegistrationErrorCode } from './utils';

// These error codes will have "detail" (e.g. the response code) displayed on
// the page.  Error codes not in this lists are for issues that are irrespective
// of the SDK's registration response (e.g. missing signing key).
const errorCodesWithDetail: RegistrationErrorCode[] = [
  'forbidden',
  'internal_server_error',
  'unauthorized',
  'url_not_found',
];

type Props = {
  errorCode: RegistrationErrorCode | undefined;
  headers: Record<string, string[]>;
  statusCode: number | undefined;
};

export function DeployFailure({ errorCode, headers, statusCode }: Props) {
  return (
    <Alert className="mb-4" severity="error">
      {createMessage(errorCode)}
      {createDetail({ errorCode, headers, statusCode })}
    </Alert>
  );
}

function createDetail({ errorCode, headers, statusCode }: Props) {
  if (!errorCode || !errorCodesWithDetail.includes(errorCode)) {
    return null;
  }

  let headersElem: JSX.Element | undefined;
  if (Object.keys(headers).length > 0) {
    headersElem = (
      <>
        <div>Headers:</div>
        {Object.entries(headers).map(([name, values]) => {
          return (
            <div className="pl-4" key={name}>
              {name}: {values.map((value) => `"${value}"`).join(', ')}
            </div>
          );
        })}
      </>
    );
  }

  let statusCodeElem: JSX.Element | undefined;
  if (statusCode) {
    statusCodeElem = <div>Status code: {statusCode}</div>;
  }

  if (!headersElem && !statusCodeElem) {
    return;
  }

  return (
    <div className="mt-2 text-slate-800">
      <div>This is from your {"app's"} response:</div>
      <pre className="w-full overflow-scroll rounded-md border border-slate-300 bg-slate-100 p-1 ">
        {statusCodeElem}
        {headersElem}
      </pre>
    </div>
  );
}

function createMessage(errorCode: RegistrationErrorCode | undefined) {
  let docsURL: string | undefined = undefined;
  let message: string;

  switch (errorCode) {
    case 'batch_size_too_large':
      docsURL = 'https://www.inngest.com/docs/reference/functions/create#configuration';
      message = 'Your function has a batch size that is too large.';
      break;
    case 'forbidden':
      message =
        'Your app rejected the request as forbidden. Is the URL behind required authentication?';
      break;
    case 'internal_server_error':
      message = 'Your app had an internal server error.';
      break;
    case 'invalid_function':
      docsURL = 'https://www.inngest.com/docs/reference/functions/create';
      message = 'Your app has at least 1 invalid function.';
      break;
    case 'invalid_signing_key':
      docsURL = 'https://www.inngest.com/docs/sdk/serve#signing-key';
      message = 'Your app is using an invalid signing key.';
      break;
    case 'missing_branch_env_name':
      docsURL =
        'https://www.inngest.com/docs/platform/environments#configuring-branch-environments';
      message = 'Your app has not specified a branch environment name.';
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
      message =
        'Your app rejected the request as unauthorized. Please disable authorization on this endpoint.';
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
    case undefined:
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
    <>
      {message} {docsMessage}
    </>
  );
}
