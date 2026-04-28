import type { MouseEventHandler } from 'react';

import { isSafeCTAURL } from '@/components/ActiveBanners/safeUrl';

const advisoryMarker = 'See more information:';

export function parseSyncErrorMessage(error: string) {
  const markerIndex = error.indexOf(advisoryMarker);
  if (markerIndex === -1) {
    return { message: error, advisoryURL: null };
  }

  const advisoryURL = error.slice(markerIndex + advisoryMarker.length).trim();
  return {
    message: error.slice(0, markerIndex).trim(),
    advisoryURL: advisoryURL && isSafeCTAURL(advisoryURL) ? advisoryURL : null,
  };
}

export function SyncErrorMessage({
  error,
  onLinkClick,
}: {
  error: string;
  onLinkClick?: MouseEventHandler<HTMLAnchorElement>;
}) {
  const { advisoryURL, message } = parseSyncErrorMessage(error);

  return (
    <>
      {message}
      {advisoryURL && (
        <>
          {' '}
          <a
            className="text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense inline underline underline-offset-2"
            href={advisoryURL}
            onClick={onLinkClick}
            rel="noopener noreferrer"
            target="_blank"
          >
            View advisory
          </a>
        </>
      )}
    </>
  );
}
