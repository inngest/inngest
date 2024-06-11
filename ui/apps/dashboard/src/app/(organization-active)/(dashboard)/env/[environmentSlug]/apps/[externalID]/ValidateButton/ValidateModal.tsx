import { useCallback, useEffect, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';

import Input from '@/components/Forms/Input';
import { type AppCheckResult } from '@/gql/graphql';
import { AccordionCard } from './AccordionCard';
import { Checks } from './Checks';
import { ConfigDetail } from './ConfigDetail';
import { HTTPInfo } from './HTTPInfo';
import { useGetAppInfo } from './getAppInfo';

type Props = {
  isOpen: boolean;
  onClose: () => void;
  url: string;
};

export function ValidateModal(props: Props) {
  const { isOpen } = props;

  const _onClose = props.onClose;
  const onClose = useCallback(() => {
    _onClose();
    setData(undefined);
    setError(undefined);
    setOverrideValue(undefined);
  }, [_onClose]);

  const [overrideValue, setOverrideValue] = useState<string>();
  const url = overrideValue ?? props.url;

  const [data, setData] = useState<AppCheckResult>();
  const [error, setError] = useState<Error>();
  const [isLoading, setIsLoading] = useState(false);

  const getAppInfo = useGetAppInfo();

  const check = useCallback(async () => {
    setIsLoading(true);

    try {
      const appCheck = await getAppInfo(url);
      setData(appCheck);
      setError(undefined);
    } catch (error) {
      setData(undefined);
      if (error instanceof Error) {
        setError(error);
      } else {
        setError(new Error('unknown error'));
      }
    } finally {
      setIsLoading(false);
    }
  }, [getAppInfo, url]);

  useEffect(() => {
    if (!data && !error && !isLoading) {
      // Load data on open
      check();
    }
  }, [check, data, error, isLoading, url]);

  return (
    <Modal className="w-[800px]" isOpen={isOpen} onClose={onClose}>
      <Modal.Header>Inspect app</Modal.Header>

      <Modal.Body>
        <p className="mb-2">Securely validate the configuration of the app at the given URL.</p>

        <p className="text-gray-500">
          The app will only return privileged information if {"it's"} using this {"environment's"}{' '}
          signing key.
        </p>

        <div className="my-4 flex flex-1 gap-4">
          <div className="grow">
            <Input
              placeholder="https://example.com/api/inngest"
              name="url"
              value={url}
              onChange={(e) => {
                setOverrideValue(e.target.value);
              }}
            />
          </div>
          <Button btnAction={check} disabled={isLoading} kind="primary" label="Retry" />
        </div>

        <hr className="my-4" />

        {error && !isLoading && <Alert severity="error">{error.message}</Alert>}

        {data && (
          <div>
            <Checks appInfo={data} />

            <AccordionCard>
              <AccordionCard.Item header="SDK configuration" value="config">
                <ConfigDetail data={data} />
              </AccordionCard.Item>

              <AccordionCard.Item header="HTTP response" value="http">
                <HTTPInfo data={data} />
              </AccordionCard.Item>
            </AccordionCard>
          </div>
        )}
      </Modal.Body>

      <Modal.Footer className="flex justify-end gap-2">
        <Button appearance="outlined" btnAction={onClose} disabled={isLoading} label="Close" />
      </Modal.Footer>
    </Modal>
  );
}
