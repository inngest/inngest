import type { Route } from 'next';
import { Link } from '@inngest/components/Link';
import { cn } from '@inngest/components/utils/classNames';

import { CardItem } from '../AppInfoCard/CardItem';

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
        <Link href={`${repoURL}/commit/${commitHash}` as Route} target="_blank" size="medium">
          <span className="truncate">{commitHash.substring(0, 7)}</span>
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
        <Link href={`${repoURL}/tree/${commitRef}` as Route} target="_blank" size="medium">
          <span className="truncate">{commitRef}</span>
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
      <Link href={repoURL as Route} target="_blank" size="medium">
        <span className="truncate">{repoURL}</span>
      </Link>
    );
  } else {
    repositoryValue = '-';
  }

  return (
    <div className={cn('border-muted bg-canvasBase overflow-hidden rounded-lg border', className)}>
      <div className="border-muted text-basis border-b px-6 py-3 text-sm font-medium">
        Commit Information
      </div>

      <dl className="flex flex-col gap-4 px-6 py-4 md:grid md:grid-cols-4">
        {/* Row 1 */}
        <CardItem className="col-span-4" detail={commitMessage} term="Commit Message" />

        {/* Row 2 */}
        <CardItem className="truncate" detail={commitAuthor} term="Commit Author" />
        <CardItem className="truncate" detail={commitRefValue} term="Commit Ref" />
        <CardItem className="truncate" detail={commitHashValue} term="Commit Hash" />

        {/* Row 3 */}
        <CardItem className="col-span-4 truncate" detail={repositoryValue} term="Repository" />
      </dl>
    </div>
  );
}
