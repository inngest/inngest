import type { Route } from 'next';
import { Link } from '@inngest/components/Link';
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

export function AppGitCard({ className, sync }: Props) {
  const { commitAuthor, commitHash, commitMessage, commitRef, repoURL } = sync;
  if (!commitAuthor && !commitHash && !commitMessage && !commitRef && !repoURL) {
    return null;
  }

  let commitHashValue;
  if (commitHash) {
    if (repoURL) {
      commitHashValue = (
        <Link href={`${repoURL}/commit/${commitHash}` as Route} internalNavigation={false}>
          {commitHash.substring(0, 7)}
        </Link>
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
        <Link href={`${repoURL}/tree/${commitRef}` as Route} internalNavigation={false}>
          {commitRef}
        </Link>
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
      <Link href={repoURL as Route} internalNavigation={false}>
        {repoURL}
      </Link>
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
      <div className="border-b border-slate-300 px-6 py-3 text-sm font-medium text-slate-600">
        Commit Information
      </div>

      <dl className="grid grow grid-cols-4 gap-4 px-6 py-4">
        {/* Row 1 */}
        <Description className="col-span-4" detail={commitMessage} term="Commit Message" />

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
      <dt className="pb-2 text-sm text-slate-400">{term}</dt>
      <dd className="text-slate-800">{detail ?? ''}</dd>
    </div>
  );
}
