import type { Route } from 'next';
import { AppDetailsCard, CardItem } from '@inngest/components/Apps/AppDetailsCard';
import { Link } from '@inngest/components/Link';

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
        <Link href={`${repoURL}/commit/${commitHash}` as Route} target="_blank" size="small">
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
        <Link href={`${repoURL}/tree/${commitRef}` as Route} target="_blank" size="small">
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
      <Link href={repoURL as Route} target="_blank" size="small">
        <span className="truncate">{repoURL}</span>
      </Link>
    );
  } else {
    repositoryValue = '-';
  }

  return (
    <AppDetailsCard className={className} title="Commit information">
      {/* Row 1 */}
      <CardItem className="col-span-4" detail={commitMessage} term="Commit message" />

      {/* Row 2 */}
      <CardItem className="truncate" detail={commitAuthor} term="Commit author" />
      <CardItem className="truncate" detail={commitRefValue} term="Commit ref" />
      <CardItem className="truncate" detail={commitHashValue} term="Commit hash" />

      {/* Row 3 */}
      <CardItem className="col-span-4 truncate" detail={repositoryValue} term="Repository" />
    </AppDetailsCard>
  );
}
