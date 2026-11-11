import { useEffect, useId, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Textarea } from '@inngest/components/Forms/Textarea';
import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import {
  RiArrowLeftLine,
  RiArrowRightLine,
  RiCheckLine,
  RiCloseLine,
  RiCloudLine,
  RiSparklingLine,
} from '@remixicon/react';

import { useTracking } from '@/hooks/useTracking';

import { DEPLOYMENT_PROMPT } from './prompt';
import { CompactDeploymentGuide } from './CompactDeploymentGuide';

type DeployAction =
  | 'opened'
  | 'prompt_copied'
  | 'manual_opened'
  | 'cloud_clicked'
  | 'docs_clicked';

export function DeployToCloudPopover() {
  const { trackEvent } = useTracking();
  function track(action: DeployAction) {
    void trackEvent('cli/dev_ui.deploy_to_cloud.action', { action });
  }

  const [open, setOpen] = useState(false);
  const [manual, setManual] = useState(false);
  const [animateSwitch, setAnimateSwitch] = useState(false);
  const manualTitle = useRef<HTMLHeadingElement>(null);
  const manualEntry = useRef<HTMLButtonElement>(null);
  const [copyState, setCopyState] = useState<
    'idle' | 'copying' | 'copied' | 'failed'
  >('idle');
  const titleID = useId();
  const descriptionID = useId();
  const panel = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (manual) manualTitle.current?.focus({ preventScroll: true });
    else manualEntry.current?.focus({ preventScroll: true });
  }, [manual]);

  useEffect(() => {
    if (copyState !== 'copied') return;
    const timer = setTimeout(() => setCopyState('idle'), 2000);
    return () => clearTimeout(timer);
  }, [copyState]);

  useEffect(() => {
    if (open && !manual && copyState === 'failed') {
      const textarea = panel.current?.querySelector('textarea');
      textarea?.focus();
      textarea?.select();
    }
  }, [open, manual, copyState]);

  async function copyPrompt() {
    if (copyState === 'copying') return;
    setCopyState('copying');
    try {
      await navigator.clipboard.writeText(DEPLOYMENT_PROMPT);
      track('prompt_copied');
      setCopyState('copied');
    } catch {
      setCopyState('failed');
    }
  }

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        setAnimateSwitch(false);
        if (nextOpen) track('opened');
        setOpen(nextOpen);
      }}
    >
      <PopoverTrigger asChild>
        <Button
          kind="secondary"
          appearance="outlined"
          icon={<RiCloudLine />}
          iconSide="left"
          label="Deploy to Cloud"
        />
      </PopoverTrigger>
      <PopoverContent
        align="end"
        sideOffset={8}
        collisionPadding={12}
        className="max-h-[min(calc(100dvh-24px),var(--radix-popover-content-available-height))] w-[300px] max-w-[calc(100vw-24px)] overflow-y-auto p-3 shadow-lg"
        aria-labelledby={titleID}
        aria-describedby={descriptionID}
      >
        <div className="mb-2 flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            {manual && (
              <Button
                kind="secondary"
                appearance="ghost"
                size="small"
                icon={<RiArrowLeftLine />}
                aria-label="Back to AI setup"
                onClick={() => {
                  setAnimateSwitch(true);
                  setManual(false);
                }}
              />
            )}
            <h2
              id={titleID}
              ref={manualTitle}
              tabIndex={-1}
              className="text-sm font-medium leading-5 outline-none"
            >
              {manual ? 'Deploy manually' : 'Deploy to Cloud'}
            </h2>
          </div>
          <PopoverClose asChild>
            <Button
              kind="secondary"
              appearance="ghost"
              size="small"
              icon={<RiCloseLine />}
              aria-label="Close Deploy to Cloud"
            />
          </PopoverClose>
        </div>
        {/* Both panels size the same grid row. Set inert on the DOM because
            React 18 drops boolean inert props. */}
        <div className={animateSwitch ? 'cloud-deploy-switch grid' : 'grid'}>
          <div
            className={`col-start-1 row-start-1 min-w-0 ${manual ? 'pointer-events-none opacity-0' : ''}`}
            aria-hidden={manual}
            ref={(node) => {
              node?.toggleAttribute('inert', manual);
            }}
          >
            <p
              id={manual ? undefined : descriptionID}
              className="text-muted mb-2 text-xs leading-4"
            >
              Use your coding agent to deploy your app and verify its Cloud
              sync.
            </p>
            <div ref={panel} className="mb-2 space-y-1.5">
              <Textarea
                aria-label="Cloud deployment prompt"
                value={DEPLOYMENT_PROMPT}
                rows={3}
                readOnly
                className="cloud-deploy-prompt block resize-none overflow-y-scroll py-2 font-mono text-xs leading-4 caret-transparent focus:outline-muted focus:-outline-offset-2"
              />
              {copyState === 'failed' && (
                <p role="status" className="text-muted text-xs">
                  Couldn’t copy. Select and copy the prompt above.
                </p>
              )}
            </div>
            <Button
              className="w-full"
              label={
                copyState === 'copied' ? 'Copied!' : 'Copy deployment prompt'
              }
              icon={
                copyState === 'copied' ? <RiCheckLine /> : <RiSparklingLine />
              }
              iconSide="left"
              disabled={copyState === 'copying'}
              onClick={copyPrompt}
            />
            <div className="mt-2 flex flex-wrap items-center gap-x-1 gap-y-1">
              <span className="text-muted text-xs">
                Want to do it yourself?
              </span>
              <Button
                ref={manualEntry}
                kind="secondary"
                appearance="ghost"
                size="small"
                label="Deploy manually"
                icon={<RiArrowRightLine />}
                iconSide="right"
                onClick={() => {
                  setAnimateSwitch(true);
                  setManual(true);
                  track('manual_opened');
                }}
              />
            </div>
          </div>
          <div
            className={`col-start-1 row-start-1 flex min-w-0 flex-col ${!manual ? 'pointer-events-none opacity-0' : ''}`}
            aria-hidden={!manual}
            ref={(node) => {
              node?.toggleAttribute('inert', !manual);
            }}
          >
            <CompactDeploymentGuide
              descriptionID={manual ? descriptionID : undefined}
              onLinkClick={track}
            />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
