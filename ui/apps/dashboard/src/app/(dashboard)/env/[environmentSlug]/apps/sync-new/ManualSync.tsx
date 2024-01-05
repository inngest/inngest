import { useMemo, useState } from 'react';
import { type Route } from 'next';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Link as InngestLink, defaultLinkStyles } from '@inngest/components/Link';
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
          Inngest allows you to host your app code on any platform then invokes your functions via
          HTTP. To deploy functions to Inngest, all you need to do is tell Inngest where to find
          them!
        </p>
        <br />
        <p>
          Since your code is hosted on another platform, you need to register where your functions
          are hosted with Inngest. For example, if you set up the serve handler (
          <Link
            className={defaultLinkStyles}
            href="https://www.inngest.com/docs/reference/serve#how-the-serve-api-handler-works"
          >
            see docs
          </Link>
          ) at /api/inngest, and your domain is https://myapp.com, you&apos;ll need to inform
          Inngest that your app is hosted at https://myapp.com/api/inngest.
        </p>
        <br />
        <p>
          After you&apos;ve set up the serve API and deployed your application, enter the URL of
          your application&apos;s serve endpoint to register your functions with Inngest.
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
      <footer className="flex items-center justify-between px-8 py-6">
        {/* To do:  create apps docs and link them here */}
        <InngestLink href="https://www.inngest.com/docs/">View Docs</InngestLink>
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
      </footer>
    </>
  );
}
