import { useMemo, useState } from 'react';
import { RocketLaunchIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { useLocalStorage } from 'react-use';
import { toast } from 'sonner';

import Modal from '@/components/Modal';
import { DOCS_URLS } from '@/utils/urls';
import { DeployFailure } from './DeployFailure';
import DeploySigningKey from './DeploySigningKey';
import { deployViaUrl, type RegistrationFailure } from './utils';

type DeployModalProps = {
  isOpen: boolean;
  onClose: () => void;
};

export default function DeployModal({ isOpen, onClose }: DeployModalProps) {
  const [failure, setFailure] = useState<RegistrationFailure>();
  const [input = '', setInput] = useLocalStorage('deploymentUrl', '');
  const [isLoading, setIsLoading] = useState(false);

  async function onClickDeploy() {
    setIsLoading(true);

    try {
      const failure = await deployViaUrl(input);
      setFailure(failure);
      if (!failure) {
        toast.success('Your app has been deployed!');
        onClose();
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
    <Modal className="flex max-w-xl flex-col gap-4" isOpen={isOpen} onClose={onClose}>
      <header className="flex flex-row items-center gap-3">
        <RocketLaunchIcon className="h-5 text-indigo-500" />
        <h2 className="text-lg font-medium">Deploy your application</h2>
      </header>
      <p>
        After you&apos;ve set up the{' '}
        <a href={`${DOCS_URLS.SERVE}?ref=app-deploy-modal`} target="_blank noreferrer">
          serve
        </a>{' '}
        API and deployed your application, enter the URL of your application&apos;s serve endpoint
        to register your functions with Inngest.
      </p>
      {/* TODO - Add CTA/info block about Vercel/Netlify integrations */}
      <DeploySigningKey />
      <div>
        <input
          className="w-full rounded-lg border px-4 py-2"
          type="text"
          placeholder="https://example.com/api/inngest"
          name="url"
          value={input}
          onChange={(e) => setInput(e.target.value)}
        />
      </div>
      {failure && !isLoading ? <DeployFailure {...failure} /> : null}
      <div className="mt-2 flex flex-row justify-end">
        <Button
          kind="primary"
          className="px-16"
          btnAction={onClickDeploy}
          disabled={disabled || isLoading}
          label="Deploy"
        />
      </div>
    </Modal>
  );
}
