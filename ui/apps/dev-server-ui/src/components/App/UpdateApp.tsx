import { useState } from 'react';
import { Input } from '@inngest/components/Forms/Input';
import { Link } from '@inngest/components/Link/Link';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { cn } from '@inngest/components/utils/classNames';
import { RiExternalLinkLine } from '@remixicon/react';
import { toast } from 'sonner';

import { useUpdateAppMutation, type GetAppsQuery } from '@/store/generated';
import isValidUrl from '@/utils/urlValidation';

export default function UpdateApp({ app }: { app: GetAppsQuery['apps'][number] }) {
  const [inputUrl, setInputUrl] = useState(app.url || '');
  const [isUrlInvalid, setUrlInvalid] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [_updateApp] = useUpdateAppMutation();
  const debouncedRequest = useDebounce(() => {
    if (isValidUrl(inputUrl)) {
      setUrlInvalid(false);
      updateApp();
    } else {
      setUrlInvalid(true);
      setIsLoading(false);
    }
  });

  async function updateApp() {
    try {
      const response = await _updateApp({
        input: {
          url: inputUrl,
          id: app.id,
        },
      });
      toast.success('The URL was successfully updated.');
      console.log('Edited app URL:', response);
    } catch (error) {
      toast.error('The URL could not be updated.');
      console.error('Error editing app:', error);
    }
    setIsLoading(false);
  }
  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    setInputUrl(e.target.value);
    setIsLoading(true);
    debouncedRequest();
  }

  return (
    <form>
      <div className="relative mb-3">
        <Input
          id="editAppUrl"
          className={cn('w-full', isLoading && 'pr-6')}
          disabled={app.autodiscovered}
          error={isUrlInvalid ? 'Please enter a valid URL' : undefined}
          value={inputUrl}
          placeholder="http://localhost:3000/api/inngest"
          onChange={handleChange}
          readOnly={app.autodiscovered}
        />
        {isLoading && <IconSpinner className="absolute right-2 top-1/3" />}
      </div>
      <Link
        size="small"
        target="_blank"
        href="https://www.inngest.com/docs/sdk/serve?ref=dev-app"
        iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
      >
        Syncing to the Dev Server
      </Link>
    </form>
  );
}
