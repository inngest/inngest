import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { NewButton } from '@inngest/components/Button';
import { NewLink } from '@inngest/components/Link';

export default function Connect({ next }: { next: () => void }) {
  // TO DO: Add interactions and pass actions as props
  return (
    <>
      <p className="text-sm">
        Inngest can setup and connect to your Neon Database automatically. Click the button below
        and we will run a few lines of code for you to make your set up easier.
      </p>

      <div className="my-6">
        <p className="mb-3">Inngest will:</p>
        <AccordionList type="multiple" defaultValue={['1']}>
          <AccordionList.Item value="1">
            <AccordionList.Trigger>Create a Postgres role for replication</AccordionList.Trigger>

            <AccordionList.Content>
              <p>
                Create a dedicated Postgres role for replicating data. The role must have the{' '}
                <code className="text-accent-xIntense text-xs">REPLICATION</code> privilege. The
                default Postgres role created with your Neon project and roles created using the
                Neon CLI, Console, or API are granted membership in the{' '}
                <code className="text-link text-xs">neon_superuser</code> role, which has the
                required <code className="text-accent-xIntense text-xs">REPLICATION</code>{' '}
                privilege.
              </p>
            </AccordionList.Content>
          </AccordionList.Item>
          <AccordionList.Item value="2">
            <AccordionList.Trigger>Grant schema access to your Postgres role</AccordionList.Trigger>
            <AccordionList.Content>
              Granting{' '}
              <code className="text-accent-xIntense text-xs">SELECT ON ALL TABLES IN SCHEMA</code>{' '}
              instead of naming the specific tables avoids having to add privileges later if you add
              tables to your publication.
            </AccordionList.Content>
          </AccordionList.Item>
          <AccordionList.Item value="3">
            <AccordionList.Trigger>Create a replication slot</AccordionList.Trigger>
            <AccordionList.Content>
              <p>
                Inngest uses the <code className="text-accent-xIntense text-xs">pgoutput</code>{' '}
                plugin in Postgres for decoding WAL changes into a logical replication stream. To
                create a replication slot called{' '}
                <code className="text-accent-xIntense text-xs">inngest_slot</code> that uses the{' '}
                <code className="text-accent-xIntense text-xs">pgoutput</code> plugin, run the
                following command on your database using your replication role:
              </p>
              <p>
                <code className="text-accent-xIntense text-xs">inngest_slot</code> is the name
                assigned to the replication slot. You will need to provide this name when you set up
                your Inngest events.
              </p>
            </AccordionList.Content>
          </AccordionList.Item>
          <AccordionList.Item value="4">
            <AccordionList.Trigger>Create a publication</AccordionList.Trigger>
            <AccordionList.Content>
              <ol>
                <li>
                  1. Add the replication identity (the method of distinguishing between rows) for
                  each table you want to replicate:
                </li>
                <li>
                  2. Create the Postgres publication. Include all tables you want to replicate as
                  part of the publication:
                </li>
              </ol>
              <p>
                The publication name is customizable. Refer to the{' '}
                <NewLink className="inline-block" size="small" href="">
                  Postgres Docs
                </NewLink>{' '}
                if you need to add or remove tables from your publication.
              </p>
            </AccordionList.Content>
          </AccordionList.Item>
        </AccordionList>
      </div>

      <NewButton label="Auto-connect" onClick={next} />
    </>
  );
}
