import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { FONT, LINE_HEIGHT, createColors, createRules } from '@inngest/components/utils/monaco';
import { isDark } from '@inngest/components/utils/theme';
import Editor, { useMonaco } from '@monaco-editor/react';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import useModifierKey from '@/hooks/useModifierKey';
import { usePortal } from '../../hooks/usePortal';
import { useSendEventMutation } from '../../store/devApi';
import { genericiseEvent } from '../../utils/events';

type SendEventModalProps = {
  data?: string | null;
  isOpen: boolean;
  onClose: () => void;
};

export default function SendEventModal({ data, isOpen, onClose }: SendEventModalProps) {
  const [dark, setDark] = useState(isDark());
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [_sendEvent, sendEventState] = useSendEventMutation();
  const portal = usePortal();
  const eventDataStr = data;

  // Define the keydown event handler
  const handleGlobalKeyDown = (event: KeyboardEvent) => {
    // Check if Ctrl or Cmd key is pressed (depending on the user's OS)
    const isCtrlCmdPressed = event.ctrlKey || event.metaKey;

    if (isCtrlCmdPressed && event.key === 'Enter') {
      // Trigger the sendEvent function
      sendEventRef.current();
    }
  };

  useEffect(() => {
    //@ts-ignore
    document.addEventListener('keydown', handleGlobalKeyDown);

    // Detach the event listener when the component unmounts
    return () => {
      //@ts-ignore
      document.removeEventListener('keydown', handleGlobalKeyDown);
    };
  }, []);

  const snippedData = useMemo(() => genericiseEvent(eventDataStr), [eventDataStr]);

  const [input, setInput] = useState(snippedData);
  useEffect(() => {
    setInput(genericiseEvent(snippedData));
  }, [eventDataStr, isOpen]);

  const pushToast = (message: string) => {
    alert(message);
  };

  const sendEvent = useCallback<() => void>(() => {
    let data: any;

    try {
      data = JSON.parse(input || '');

      if (typeof data.id !== 'string') {
        data.id = ulid();
      }

      if (!data.ts || typeof data.ts !== 'number') {
        data.ts = Date.now();
      }
    } catch (err) {
      return pushToast('Event payload could not be parsed as JSON.');
    }

    if (!data.name) {
      return pushToast('Event payload must contain a name.');
    }

    if (typeof data.name !== 'string') {
      return pushToast(
        "Event payload name must be a string, ideally in the format 'scope/subject.verb'."
      );
    }

    if (data.data && typeof data.data !== 'object') {
      return pushToast('Event payload data must be an object if defined.');
    }

    if (data.user && typeof data.user !== 'object') {
      return pushToast('Event payload user must be an object if defined.');
    }

    _sendEvent(data)
      .unwrap()
      .then(() => {
        toast.success('The event was successfully added.');
        onClose();
      });
  }, [_sendEvent, input]);

  const monaco = useMonaco();

  const sendEventRef = useRef(sendEvent);
  useEffect(() => {
    sendEventRef.current = sendEvent;
  }, [sendEvent]);

  useEffect(() => {
    // We don't have a DOM ref until we're rendered, so check for dark theme parent classes then
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
            type: 'object',
            properties: {
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
            },
            required: ['name'],
          },
        },
      ],
    });
  }, [monaco, dark]);

  return portal(
    <Modal isOpen={isOpen} onClose={onClose} className="w-full max-w-5xl">
      <Modal.Header description="Send an event manually by filling or pasting a payload">
        Send Event
      </Modal.Header>
      <Modal.Body>
        <div
          className="border-subtle relative flex h-[20rem] w-full flex-col overflow-hidden rounded-md border"
          ref={wrapperRef}
        >
          <div className="border-subtle flex items-center justify-between border-b">
            <p className=" text-subtle px-5 py-2.5 text-sm">Payload</p>
          </div>
          {monaco ? (
            <Editor
              defaultLanguage="json"
              value={input ?? '{}'}
              onChange={(value) => setInput(value || '')}
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
      </Modal.Body>
      <Modal.Footer className="flex justify-end gap-2">
        <Button kind="secondary" label="Cancel" appearance="outlined" onClick={onClose} />
        <Button
          kind="primary"
          disabled={sendEventState.isLoading}
          label="Send event"
          onClick={() => sendEvent()}
          keys={[useModifierKey(), '↵']}
        />
      </Modal.Footer>
    </Modal>
  );
}
