import { classNames } from '@inngest/components/utils/classNames';

type Props = {
  className?: string;
  sync: {
    commitAuthor: string | null;
    commitHash: string | null;
    commitMessage: string | null;
    commitRef: string | null;
    repoURL: string | null;
  };
};

export function GitCard({ className, sync }: Props) {
  const { commitAuthor, commitHash, commitMessage, commitRef, repoURL } = sync;
  if (!commitAuthor && !commitHash && !commitMessage && !commitRef && !repoURL) {
    return null;
  }

  let commitHashValue;
  if (commitHash) {
    if (repoURL) {
      commitHashValue = (
        <a
          className="text-indigo-600 hover:underline"
          href={`${repoURL}/commit/${commitHash}`}
          target="_blank"
        >
          {commitHash.substring(0, 7)}
        </a>
      );
    } else {
      commitHashValue = commitHash.substring(0, 7);
    }
  } else {
    commitHashValue = '-';
  }

  let commitRefValue;
  if (commitRef) {
    if (repoURL) {
      commitRefValue = (
        <a
          className="text-indigo-600 hover:underline"
          href={`${repoURL}/tree/${commitRef}`}
          target="_blank"
        >
          {commitRef}
        </a>
      );
    } else {
      commitRefValue = commitRef;
    }
  } else {
    commitRefValue = '-';
  }

  let repositoryValue;
  if (repoURL) {
    repositoryValue = (
      <a className="text-indigo-600 hover:underline" href={repoURL} target="_blank">
        {repoURL}
      </a>
    );
  } else {
    repositoryValue = '-';
  }

  return (
    <div
      className={classNames(
        'overflow-hidden rounded-lg border border-slate-300 bg-white',
        className
      )}
    >
      <div className="border-b border-slate-300 px-4 py-2">Commit Info</div>

      <dl className="grid grow grid-cols-4 gap-4 p-4">
        {/* Row 1 */}
        <Description
          className="col-span-4"
          detail={
            <div>
              <code className="whitespace-pre-line text-sm">{commitMessage}</code>
            </div>
          }
          term="Commit Message"
        />

        {/* Row 2 */}
        <Description className="truncate" detail={commitAuthor} term="Commit Author" />
        <Description className="truncate" detail={commitRefValue} term="Commit Ref" />
        <Description className="truncate" detail={commitHashValue} term="Commit Hash" />

        {/* Row 3 */}
        <Description className="col-span-4 truncate" detail={repositoryValue} term="Repository" />
      </dl>
    </div>
  );
}

function Description({
  className,
  detail,
  term,
}: {
  className?: string;
  detail: React.ReactNode;
  term: string;
}) {
  return (
    <div className={className}>
      <dt className="text-xs text-slate-600">{term}</dt>
      <dd>{detail ?? ''}</dd>
    </div>
  );
}
