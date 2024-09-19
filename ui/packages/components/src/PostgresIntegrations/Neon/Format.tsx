import { NewButton } from '@inngest/components/Button';
import { NewLink } from '@inngest/components/Link';

export default function NeonFormat({ next }: { next: () => void }) {
  // TO DO: Add interactions and pass actions as props
  return (
    <>
      <p className="text-sm">
        Enabling logical replication modifies the Postgres{' '}
        <code className="text-accent-xIntense text-xs">wal_level</code> configuration parameter,
        changing it from <code className="text-accent-xIntense text-xs">replica</code> to{' '}
        <code className="text-accent-xIntense text-xs">logical</code> for all databases in your Neon
        project. Once the <code className="text-accent-xIntense text-xs">wal_level</code> setting is
        changed to <code className="text-accent-xIntense text-xs">logical</code>, it cannot be
        reverted. Enabling logical replication also restarts all computes in your Neon project,
        meaning active connections will be dropped and have to reconnect.
      </p>
      <NewLink
        size="small"
        href="https://neon.tech/docs/guides/logical-replication-concepts#write-ahead-log-wal"
      >
        Learn more about WAL level
      </NewLink>

      <div className="my-6">
        <p className="mb-3">To enable logical replication in Neon:</p>
        <ol className="list-decimal pl-10 text-sm leading-8">
          <li>Select your project in the Neon Console.</li>
          <li>
            On the Neon <span className="text-medium">Dashboard</span>, select{' '}
            <span className="text-medium">Settings</span>.
          </li>
          <li>
            Select <span className="text-medium">Logical Replication</span>.
          </li>
          <li>
            Click <span className="text-medium">Enable</span> to enable logical replication.
          </li>
        </ol>
      </div>

      <NewButton label="Next" onClick={next} />
    </>
  );
}
