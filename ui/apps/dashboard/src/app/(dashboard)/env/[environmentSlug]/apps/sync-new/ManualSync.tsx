import { useMemo, useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Code } from '@inngest/components/Code';
import { Link } from '@inngest/components/Link';
import { useLocalStorage } from 'react-use';
import { toast } from 'sonner';

import Input from '@/components/Forms/Input';
import { DeployFailure } from '../../deploys/DeployFailure';
import DeploySigningKey from '../../deploys/DeploySigningKey';
import { deployViaUrl, type RegistrationFailure } from '../../deploys/utils';

type Props = {
  appsURL: Route;
};

export default function ManualSync({ appsURL }: Props) {
  const [input = '', setInput] = useLocalStorage('deploymentUrl', '');
  const [failure, setFailure] = useState<RegistrationFailure>();
  const [isLoading, setIsLoading] = useState(false);
  const router = useRouter();

  async function onClickDeploy() {
    setIsLoading(true);

    try {
      const failure = await deployViaUrl(input);
      setFailure(failure);
      if (!failure) {
        toast.success('App Successfuly Synced');
        router.push(appsURL);
      }
    } catch {
      setFailure({
        errorCode: undefined,
        headers: {},
        statusCode: undefined,
      });
    } finally {
      setIsLoading(false);
    }
  }

  /**
   * Disable the button if the URL isn't valid
   */
  const disabled = useMemo(() => {
    try {
      new URL(input);
      return false;
    } catch {
      return true;
    }
  }, [input]);

  return (
    <>
      <div className="border-b border-slate-200 p-8">
        <p>
          To integrate your code hosted on another platform with Inngest, you need to inform Inngest
          about the location of your app and functions.
        </p>
        <br />
        <p>
          For example, imagine that your <Code>serve()</Code> handler (
          <Link
            showIcon={false}
            href="https://www.inngest.com/docs/reference/serve#how-the-serve-api-handler-works"
          >
            see docs
          </Link>
          ) is located at /api/inngest, and your domain is myapp.com. In this scenario, you&apos;ll
          need to inform Inngest that your apps and functions are hosted at
          https://myapp.com/api/inngest.
        </p>
        <br />
        <p>
          After you&apos;ve set up the serve API and deployed your code,{' '}
          <span className="font-semibold">
            enter the URL of your project&apos;s serve endpoint to sync your app with Inngest
          </span>
          . Verify that you assigned the signing key below to the {' '}
          <Code>INNGEST_SIGNING_KEY</Code> environment variable:
        </p>
        <DeploySigningKey className="py-6" />
        <div className="border-t border-slate-200">
          <label htmlFor="url" className="my-2 block text-slate-500">
            App URL
          </label>
          <Input
            placeholder="https://example.com/api/inngest"
            name="url"
            value={input}
            onChange={(e) => setInput(e.target.value)}
          />
          {failure && !isLoading ? (
            <div className="mt-2">
              <DeployFailure {...failure} />
            </div>
          ) : null}
        </div>
      </div>
      <div className="flex items-center justify-between px-8 py-6">
        <Link href="https://www.inngest.com/docs/apps/cloud">View Docs</Link>
        <div className="flex items-center gap-3">
          <Button
            label="Cancel"
            btnAction={() => {
              router.push(appsURL);
            }}
            appearance="outlined"
          />
          <Button
            label="Sync App"
            btnAction={onClickDeploy}
            kind="primary"
            disabled={disabled}
            loading={isLoading}
          />
        </div>
      </div>
    </>
  );
}
