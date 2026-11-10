import { useCallback, useEffect, useId, useRef, useState } from 'react';
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
  | 'prompt_viewed'
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
  const [expanded, setExpanded] = useState(false);
  const [prompt, setPrompt] = useState(DEPLOYMENT_PROMPT);
  const [copyState, setCopyState] = useState<
    'idle' | 'copying' | 'copied' | 'failed'
  >('idle');
  const panelID = useId();
  const titleID = useId();
  const descriptionID = useId();
  const panel = useRef<HTMLDivElement>(null);
  const copyAttempt = useRef(0);
  const [contentHeight, setContentHeight] = useState<number>();
  const contentObserver = useRef<ResizeObserver | null>(null);
  const measureContent = useCallback((node: HTMLDivElement | null) => {
    contentObserver.current?.disconnect();
    if (!node) return;
    const updateHeight = () => setContentHeight(node.offsetHeight);
    updateHeight();
    contentObserver.current = new ResizeObserver(updateHeight);
    contentObserver.current.observe(node);
  }, []);

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
    if (open && expanded && copyState === 'failed') {
      const textarea = panel.current?.querySelector('textarea');
      textarea?.focus();
      textarea?.select();
    }
  }, [open, expanded, copyState]);

  async function copyPrompt() {
    if (copyState === 'copying' || !prompt.trim()) return;
    const attempt = ++copyAttempt.current;
    setCopyState('copying');
    try {
      await navigator.clipboard.writeText(prompt);
      track('prompt_copied');
      if (attempt === copyAttempt.current) setCopyState('copied');
    } catch {
      if (attempt !== copyAttempt.current) return;
      setCopyState('failed');
      setExpanded(true);
    }
  }

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        setAnimateSwitch(false);
        if (nextOpen) track('opened');
        else setContentHeight(undefined);
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
        className="max-h-[min(calc(100dvh-24px),var(--radix-popover-content-available-height))] w-[360px] max-w-[calc(100vw-24px)] overflow-y-auto p-5 shadow-lg"
        aria-labelledby={titleID}
        aria-describedby={manual ? undefined : descriptionID}
      >
        <div className="mb-2 flex items-center justify-between">
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
              className="font-medium outline-none"
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
        <div
          className="cloud-deploy-viewport"
          style={{ height: contentHeight }}
        >
          <div
            key={manual ? 'manual' : 'agent'}
            ref={measureContent}
            className={`flow-root ${animateSwitch ? (manual ? 'cloud-deploy-forward' : 'cloud-deploy-back') : ''}`}
          >
            {manual ? (
              <CompactDeploymentGuide onLinkClick={track} />
            ) : (
              <>
                <p id={descriptionID} className="text-muted mb-5 text-sm">
                  Use your coding agent to deploy your app and verify its Cloud
                  sync.
                </p>
                <Button
                  className="w-full"
                  label={
                    copyState === 'copied'
                      ? 'Copied!'
                      : 'Copy deployment prompt'
                  }
                  icon={
                    copyState === 'copied' ? (
                      <RiCheckLine />
                    ) : (
                      <RiSparklingLine />
                    )
                  }
                  iconSide="left"
                  disabled={!prompt.trim() || copyState === 'copying'}
                  onClick={copyPrompt}
                />
                <div className="mt-1.5 flex justify-center">
                  <Button
                    kind="primary"
                    appearance="ghost"
                    size="small"
                    label={expanded ? 'Hide prompt' : 'View prompt'}
                    aria-expanded={expanded}
                    aria-controls={panelID}
                    onClick={() => {
                      if (!expanded) track('prompt_viewed');
                      setExpanded(!expanded);
                    }}
                  />
                </div>
                {expanded && (
                  <div id={panelID} ref={panel} className="mt-4 space-y-1.5">
                    <p className="text-muted text-xs">
                      Cloud deployment prompt
                    </p>
                    <Textarea
                      aria-label="Cloud deployment prompt"
                      value={prompt}
                      rows={7}
                      className="bg-disabled text-basis border-transparent font-mono text-xs"
                      onChange={(value) => {
                        copyAttempt.current++;
                        setPrompt(value);
                        setCopyState('idle');
                      }}
                    />
                  </div>
                )}
                {copyState === 'failed' && (
                  <p role="status" className="text-muted mt-2 text-xs">
                    Couldn’t copy. Select and copy the prompt above.
                  </p>
                )}
                <div className="mt-4 flex flex-wrap items-center gap-x-1 gap-y-1">
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
              </>
            )}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
