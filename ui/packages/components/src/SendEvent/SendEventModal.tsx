import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import TabCards from '@inngest/components/TabCards/TabCards';
import { FONT, LINE_HEIGHT, createColors, createRules } from '@inngest/components/utils/monaco';
import { isDark } from '@inngest/components/utils/theme';
import Editor, { useMonaco } from '@monaco-editor/react';
import { toast } from 'sonner';

import { CodeViewer } from './CodeViewer';
import { copyToClipboard, type EventPayload } from './utils';

function KeyboardShortcut({ keys }: { keys: string[] }) {
  const [isMac, setIsMac] = useState(false);

  useEffect(() => {
    const userAgent = navigator.userAgent.toUpperCase();
    setIsMac(userAgent.indexOf('MAC') >= 0);
  }, []);

  const renderKey = (key: string) => {
    const normalizedKey = key.toLowerCase();

    if (normalizedKey === 'cmd' || normalizedKey === 'ctrl') {
      return isMac && normalizedKey === 'cmd'
        ? '⌘'
        : !isMac && normalizedKey === 'ctrl'
        ? 'Ctrl'
        : null;
    }
    if (normalizedKey === 'enter') {
      return '⏎';
    }
    return key;
  };

  const renderedKeys = keys.map(renderKey).filter(Boolean);

  return (
    <div className="flex items-center gap-0.5 rounded bg-white/20 px-1 py-0.5 text-xs">
      {renderedKeys.map((key, index) => (
        <span key={index}>{key}</span>
      ))}
    </div>
  );
}

export interface SendEventConfig {
  sendEvent: (payload: EventPayload | EventPayload[]) => Promise<void>;
  generateSDKCode: (payload: EventPayload | EventPayload[]) => string;
  generateCurlCode: (payload: EventPayload | EventPayload[]) => string;

  ui?: {
    modalTitle?: string;
    sendButtonLabel?: string;
    isLoading?: boolean;
  };

  usePortal?: () => (element: React.ReactElement) => React.ReactElement;
  processInitialData?: (data?: string | null) => string;
}

export interface SharedSendEventModalProps {
  data?: string | null;
  isOpen: boolean;
  onClose: () => void;
  config: SendEventConfig;
}

type Tab = {
  id: string;
  label: string;
  buttonLabel: string;
  buttonAction: () => void;
};

export function SendEventModal({ data, isOpen, onClose, config }: SharedSendEventModalProps) {
  const [dark, setDark] = useState(isDark());
  const [activeTab, setActiveTab] = useState('editor');
  const [isLoading, setIsLoading] = useState(false);
  const [sdkCode, setSdkCode] = useState('');
  const [curlCode, setCurlCode] = useState('');
  const [sdkDirty, setSdkDirty] = useState(false);
  const [curlDirty, setCurlDirty] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const monaco = useMonaco();
  const lastGeneratedSdkRef = useRef('');
  const lastGeneratedCurlRef = useRef('');

  // Use environment-specific hooks if provided
  const portal = config.usePortal?.() || ((element) => element);

  const [payload, setPayload] = useState<EventPayload>({ name: '', data: {} });
  const [jsonInput, setJsonInput] = useState('{}');

  useEffect(() => {
    const initialData = config.processInitialData?.(data) || data || '{}';
    try {
      const parsedPayload = JSON.parse(initialData);
      setPayload(parsedPayload);
      setJsonInput(JSON.stringify(parsedPayload, null, 2));
    } catch {
      setPayload({ name: '', data: {} });
      setJsonInput(initialData);
    }
  }, [data, isOpen, config.processInitialData]);

  // Update payload when JSON editor changes
  const handleJSONChange = useCallback((value: string | undefined) => {
    if (!value) return;

    setJsonInput(value); // Always update the input immediately

    try {
      const parsedPayload = JSON.parse(value);
      setPayload(parsedPayload); // Only update payload if valid JSON
    } catch {
      // Invalid JSON - don't update payload, but let user keep typing
    }
  }, []);

  // Keep generated code in sync with payload without overwriting user edits.
  useEffect(() => {
    const newSdk = config.generateSDKCode(payload);
    const newCurl = config.generateCurlCode(payload);
    const prevSdk = lastGeneratedSdkRef.current;
    const prevCurl = lastGeneratedCurlRef.current;

    lastGeneratedSdkRef.current = newSdk;
    lastGeneratedCurlRef.current = newCurl;

    if (!sdkDirty || sdkCode === prevSdk) {
      setSdkCode(newSdk);
      setSdkDirty(false);
    }

    if (!curlDirty || curlCode === prevCurl) {
      setCurlCode(newCurl);
      setCurlDirty(false);
    }
  }, [
    payload,
    config.generateSDKCode,
    config.generateCurlCode,
    sdkDirty,
    curlDirty,
    sdkCode,
    curlCode,
  ]);

  const preparePayloadForSending = useCallback((rawPayload: EventPayload | EventPayload[]) => {
    const events = Array.isArray(rawPayload) ? rawPayload : [rawPayload];
    return events;
  }, []);

  const sendEvent = useCallback<() => void>(async () => {
    const events = Array.isArray(payload) ? payload : [payload];
    for (const event of events) {
      if (!event.name) {
        toast.error('Each event payload must contain a name.');
        return;
      }
      if (typeof event.name !== 'string') {
        toast.error(
          "Each event payload name must be a string, ideally in the format 'scope/subject.verb'."
        );
        return;
      }
      if (event.data && typeof event.data !== 'object') {
        toast.error('Each event payload data must be an object if defined.');
        return;
      }
      if (event.user && typeof event.user !== 'object') {
        toast.error('Each event payload user must be an object if defined.');
        return;
      }
    }

    try {
      setIsLoading(true);

      const payloadToSend = preparePayloadForSending(payload);
      await config.sendEvent(payloadToSend);
      const message = Array.isArray(payload)
        ? `${payload.length} events were successfully added.`
        : 'The event was successfully added.';

      toast.success(message);
      onClose();
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
      toast.error(`Failed to send event: ${errorMessage}`);
      console.error('Send event error:', error);
    } finally {
      setIsLoading(false);
    }
  }, [config.sendEvent, payload, onClose, preparePayloadForSending]);

  const handleCopySDK = useCallback(async () => {
    await copyToClipboard(sdkCode || config.generateSDKCode(payload));
  }, [config.generateSDKCode, payload, sdkCode]);

  const handleCopyCurl = useCallback(async () => {
    await copyToClipboard(curlCode || config.generateCurlCode(payload));
  }, [config.generateCurlCode, payload, curlCode]);

  const eventProperties = {
    name: {
      type: 'string',
      description:
        "A unique identifier for the event. The recommended format is scope/subject.verb, e.g. 'app/user.created' or 'stripe/payment.succeeded'.",
    },
    data: {
      type: 'object',
      additionalProperties: true,
      description: 'Any data pertinent to the event.',
    },
    user: {
      type: 'object',
      additionalProperties: true,
      description:
        "Any user data associated with the event. All fields ending in '_id' will be used to attribute the event to a particular user.",
    },
    ts: {
      type: 'number',
      multipleOf: 1,
      minimum: 0,
      description:
        'An integer representing the milliseconds since the unix epoch at which this event occured. If omitted, the current time will be used.',
    },
  };

  const sendEventRef = useRef(sendEvent);
  useEffect(() => {
    sendEventRef.current = sendEvent;
  }, [sendEvent]);

  useEffect(() => {
    if (wrapperRef.current) {
      setDark(isDark(wrapperRef.current));
    }
  });

  useEffect(() => {
    if (!monaco) {
      return;
    }

    monaco.editor.defineTheme('inngest-theme', {
      base: dark ? 'vs-dark' : 'vs',
      inherit: true,
      rules: dark ? createRules(true) : createRules(false),
      colors: dark ? createColors(true) : createColors(false),
    });

    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
      validate: true,
      schemas: [
        {
          uri: 'https://inngest.com/event-schema.json',
          fileMatch: ['*'],
          schema: {
            oneOf: [
              {
                type: 'object',
                properties: eventProperties,
                required: ['name'],
              },
              {
                type: 'array',
                items: {
                  type: 'object',
                  properties: eventProperties,
                  required: ['name'],
                },
              },
            ],
          },
        },
      ],
    });
  }, [monaco, dark]);

  const tabs: Tab[] = useMemo(() => {
    return [
      {
        id: 'editor',
        label: 'JSON Editor',
        buttonLabel: config.ui?.sendButtonLabel || 'Send event',
        buttonAction: sendEvent,
      },
      {
        id: 'sdk',
        label: 'SDK',
        buttonLabel: 'Copy Code',
        buttonAction: handleCopySDK,
      },
      {
        id: 'curl',
        label: 'cURL',
        buttonLabel: 'Copy Code',
        buttonAction: handleCopyCurl,
      },
    ];
  }, [config.ui?.sendButtonLabel, sendEvent, handleCopySDK, handleCopyCurl]);

  const activeTabData = (tabs.find((tab) => tab.id === activeTab) ?? tabs[0])!;

  useEffect(() => {
    const handleGlobalKeyDown = (event: globalThis.KeyboardEvent) => {
      const isCtrlCmdPressed = event.ctrlKey || event.metaKey;

      if (isCtrlCmdPressed && event.key === 'Enter') {
        activeTabData.buttonAction();
      }
    };

    document.addEventListener('keydown', handleGlobalKeyDown);

    return () => {
      document.removeEventListener('keydown', handleGlobalKeyDown);
    };
  }, [activeTabData]);

  return portal(
    <Modal isOpen={isOpen} onClose={onClose} className="w-full max-w-5xl">
      <Modal.Body>
        <TabCards value={activeTab} onValueChange={setActiveTab}>
          <div className="items-top flex justify-between">
            <h2 className="text-basis text-xl">{config.ui?.modalTitle || 'Send Event'}</h2>
            {tabs.length > 1 && (
              <TabCards.ButtonList>
                {tabs.map((tab) => (
                  <TabCards.Button key={tab.id} value={tab.id}>
                    {tab.label}
                  </TabCards.Button>
                ))}
              </TabCards.ButtonList>
            )}
          </div>

          <TabCards.Content value="editor" className="p-0">
            <div
              className="border-subtle relative flex h-[20rem] w-full flex-col overflow-hidden rounded-md border"
              ref={wrapperRef}
            >
              <div className="border-subtle flex items-center justify-between border-b">
                <p className="text-subtle px-5 py-2.5 text-sm">Payload</p>
              </div>
              {monaco ? (
                <Editor
                  defaultLanguage="json"
                  value={jsonInput}
                  onChange={handleJSONChange}
                  theme="inngest-theme"
                  onMount={(editor) => {
                    editor.addAction({
                      id: 'sendInngestEvent',
                      label: 'Send Inngest Event',
                      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
                      contextMenuGroupId: '2_execution',
                      run: () => {
                        sendEventRef.current();
                      },
                    });
                  }}
                  options={{
                    minimap: {
                      enabled: false,
                    },
                    lineNumbers: 'on',
                    extraEditorClassName: '',
                    contextmenu: false,
                    inlayHints: {
                      enabled: 'on',
                    },
                    scrollBeyondLastLine: false,
                    wordWrap: 'on',
                    fontFamily: FONT.font,
                    fontSize: FONT.size,
                    fontWeight: 'light',
                    lineHeight: LINE_HEIGHT,
                    padding: {
                      top: 10,
                      bottom: 10,
                    },
                  }}
                />
              ) : null}
            </div>
          </TabCards.Content>

          <TabCards.Content value="sdk" className="p-0">
            <CodeViewer
              code={sdkCode}
              language="javascript"
              onChange={(value) => {
                setSdkDirty(true);
                setSdkCode(value || '');
              }}
            />
          </TabCards.Content>

          <TabCards.Content value="curl" className="p-0">
            <CodeViewer
              code={curlCode}
              language="bash"
              onChange={(value) => {
                setCurlDirty(true);
                setCurlCode(value || '');
              }}
            />
          </TabCards.Content>
        </TabCards>
      </Modal.Body>
      <Modal.Footer className="flex justify-end gap-2">
        <Button kind="secondary" label="Cancel" appearance="outlined" onClick={onClose} />
        <Button
          kind="primary"
          disabled={isLoading || (config.ui?.isLoading && activeTab === 'editor')}
          label={
            activeTab === 'editor' ? (
              <div className="flex items-center gap-2">
                <span>{activeTabData.buttonLabel}</span>
                <KeyboardShortcut keys={['cmd', 'ctrl', 'enter']} />
              </div>
            ) : (
              activeTabData.buttonLabel
            )
          }
          onClick={activeTabData.buttonAction}
        />
      </Modal.Footer>
    </Modal>
  );
}
