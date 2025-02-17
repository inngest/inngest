import { useCallback, useEffect, useState } from 'react';
import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';

import { type AppCheckResult } from '@/gql/graphql';
import { Checks } from './Checks';
import { ConfigDetail } from './ConfigDetail';
import { HTTPInfo } from './HTTPInfo';
import { useGetAppInfo } from './getAppInfo';

type Props = {
  isOpen: boolean;
  onClose: () => void;

  /** If set, the modal will automatically perform a check when it opens. */
  initialURL?: string;
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
  const url = overrideValue ?? props.initialURL;

  const [data, setData] = useState<AppCheckResult>();
  const [error, setError] = useState<Error>();
  const [isLoading, setIsLoading] = useState(false);

  const getAppInfo = useGetAppInfo();

  const check = useCallback(
    async (url: string) => {
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
    },
    [getAppInfo]
  );

  useEffect(() => {
    if (!data && !error && !isLoading && props.initialURL) {
      // Load data on open
      check(props.initialURL);
    }
  }, [check, data, error, isLoading, props.initialURL]);

  return (
    <Modal className="w-[800px]" isOpen={isOpen} onClose={onClose}>
      <Modal.Header>App diagnostics</Modal.Header>

      <Modal.Body>
        <p className="text-basis mb-2">
          Securely validate the configuration of the app at the given URL.
        </p>

        <p className="text-muted text-sm">
          Note: The app will only return privileged information if {"it's"} using this{' '}
          {"environment's"} signing key.
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
          <Button
            onClick={() => {
              if (!url) {
                return;
              }
              check(url);
            }}
            disabled={isLoading || !url}
            kind="primary"
            label="Check"
          />
        </div>

        <hr className="border-subtle my-4" />

        {error && !isLoading && (
          <Alert severity="error" className="text-sm">
            {error.message}
          </Alert>
        )}

        {data && (
          <div>
            <Checks appInfo={data} />

            <AccordionList type="multiple" defaultValue={[]}>
              <AccordionList.Item value="config">
                <AccordionList.Trigger className="text-sm">SDK configuration</AccordionList.Trigger>
                <AccordionList.Content className="px-9">
                  <ConfigDetail data={data} />
                </AccordionList.Content>
              </AccordionList.Item>
              <AccordionList.Item value="http">
                <AccordionList.Trigger className="text-sm">HTTP response</AccordionList.Trigger>
                <AccordionList.Content className="px-9">
                  <HTTPInfo data={data} />
                </AccordionList.Content>
              </AccordionList.Item>
            </AccordionList>
          </div>
        )}
      </Modal.Body>

      <Modal.Footer className="flex justify-end gap-2">
        <Button
          appearance="outlined"
          kind="secondary"
          onClick={onClose}
          disabled={isLoading}
          label="Close"
        />
      </Modal.Footer>
    </Modal>
  );
}
