import AppDetailsCard from "@inngest/components/Apps/AppDetailsCard";
import { Link } from "@inngest/components/Link/NewLink";

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

export const AppGitCard = ({ className, sync }: Props) => {
  const { commitAuthor, commitHash, commitMessage, commitRef, repoURL } = sync;
  if (
    !commitAuthor &&
    !commitHash &&
    !commitMessage &&
    !commitRef &&
    !repoURL
  ) {
    return null;
  }

  let commitHashValue;
  if (commitHash) {
    if (repoURL) {
      commitHashValue = (
        <Link
          href={`${repoURL}/commit/${commitHash}`}
          target="_blank"
          size="small"
        >
          <span className="truncate">{commitHash.substring(0, 7)}</span>
        </Link>
      );
    } else {
      commitHashValue = commitHash.substring(0, 7);
    }
  } else {
    commitHashValue = "-";
  }

  let commitRefValue;
  if (commitRef) {
    if (repoURL) {
      commitRefValue = (
        <Link
          href={`${repoURL}/tree/${commitRef}`}
          target="_blank"
          size="small"
        >
          <span className="truncate">{commitRef}</span>
        </Link>
      );
    } else {
      commitRefValue = commitRef;
    }
  } else {
    commitRefValue = "-";
  }

  let repositoryValue;
  if (repoURL) {
    repositoryValue = (
      <Link href={repoURL} target="_blank" size="small">
        <span className="truncate">{repoURL}</span>
      </Link>
    );
  } else {
    repositoryValue = "-";
  }

  return (
    <AppDetailsCard className={className} title="Commit information">
      <AppDetailsCard.Item
        className="col-span-4"
        detail={commitMessage}
        term="Commit message"
      />

      <AppDetailsCard.Item
        className="truncate"
        detail={commitAuthor}
        term="Commit author"
      />
      <AppDetailsCard.Item
        className="truncate"
        detail={commitRefValue}
        term="Commit ref"
      />
      <AppDetailsCard.Item
        className="truncate"
        detail={commitHashValue}
        term="Commit hash"
      />

      <AppDetailsCard.Item
        className="col-span-4 truncate"
        detail={repositoryValue}
        term="Repository"
      />
    </AppDetailsCard>
  );
};
