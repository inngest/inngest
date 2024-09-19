import { NewButton } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { NewLink } from '@inngest/components/Link';

export default function NeonAuth({ next }: { next: () => void }) {
  // TO DO: Add interactions and pass actions as props
  return (
    <>
      <p className="text-sm">
        Inngest needs to be authorized with your postgres credentials to set up replication slots,
        publications, and a new user that subscribes to updates. Note that your admin credentials
        will not be stored and are only used for setup.
      </p>
      <NewLink size="small" href="https://neon.tech/docs/connect/connect-securely">
        Learn more about postgres credentials
      </NewLink>
      <form className="py-6">
        <label className="pb-2 text-sm">
          Please paste your admin postgres credentials in the field below to continue.
        </label>
        <div className="flex items-center justify-between gap-1">
          <div className="w-full">
            <Input placeholder="eg: postgresql://neondb_owner:6sFm9owfZqSk@a5hly6e1.useast-2.aws.tech" />
          </div>
          <NewButton label="Verify" />
        </div>
      </form>
      <NewButton label="Next" onClick={next} />
    </>
  );
}
