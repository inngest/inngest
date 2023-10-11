import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import Editor, { useMonaco } from '@monaco-editor/react';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import Button from '@/components/Button/Button';
import Modal from '@/components/Modal';
import useModifierKey from '@/hooks/useModifierKey';
import { usePortal } from '../../hooks/usePortal';
import { useSendEventMutation } from '../../store/devApi';
import { genericiseEvent } from '../../utils/events';

export default function SendEventModal({ data, isOpen, onClose }) {
  const [_sendEvent, sendEventState] = useSendEventMutation();
  const portal = usePortal();
  const eventDataStr = data;

    // Define the keydown event handler
    const handleGlobalKeyDown = (event) => {
      // Check if Ctrl or Cmd key is pressed (depending on the user's OS)
      const isCtrlCmdPressed = event.ctrlKey || event.metaKey;

      if (isCtrlCmdPressed && event.key === 'Enter') {
        // Trigger the sendEvent function
        sendEventRef.current();
      }
    };

    useEffect(() => {
      document.addEventListener('keydown', handleGlobalKeyDown);

      // Detach the event listener when the component unmounts
      return () => {
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
        "Event payload name must be a string, ideally in the format 'scope/subject.verb'.",
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
    if (!monaco) {
      return;
    }

    monaco.editor.defineTheme('inngest-theme', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        {
          token: 'delimiter.bracket.json',
          foreground: 'cbd5e1', //slate-300
        },
        {
          token: 'string.key.json',
          foreground: '818cf8', //indigo-400
        },
        {
          token: 'number.json',
          foreground: 'fbbf24', //amber-400
        },
        {
          token: 'string.value.json',
          foreground: '6ee7b7', //emerald-300
        },
        {
          token: 'keyword.json',
          foreground: 'f0abfc', //fuschia-300
        },
      ],
      colors: {
        'editor.background': '#1e293b4d', // slate-800/40
        'editor.lineHighlightBorder': '#cbd5e11a', // slate-300/10
        'editorIndentGuide.background': '#cbd5e133', // slate-300/20
        'editorIndentGuide.activeBackground': '#cbd5e14d', // slate-300/30
        'editorLineNumber.foreground': '#cbd5e14d', // slate-300/30
        'editorLineNumber.activeForeground': '#CBD5E1', // slate-300
      },
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
  }, [monaco]);

  return portal(
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Send Event"
      description="Send an event manually by filling or pasting a payload"
      className="max-w-5xl w-full"
    >
      <div className="m-6">
        <div className="relative w-full h-[20rem] flex flex-col rounded-lg overflow-hidden">
          <div className="items-center bg-slate-800/40 shadow border-b border-slate-700/20 flex justify-between">
            <p className=" text-slate-400 text-xs px-5 py-4">Payload</p>
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
                fontFamily: 'RobotoMono',
                fontSize: 13,
                fontWeight: 'light',
                lineHeight: 26,
                padding: {
                  top: 10,
                  bottom: 10,
                },
              }}
            />
          ) : null}
        </div>
      </div>
      <div className="flex items-center justify-between p-6 border-t border-slate-800">
        <Button label="Cancel" appearance="outlined" btnAction={onClose} />
        <Button
          disabled={sendEventState.isLoading}
          label="Send Event"
          btnAction={() => sendEvent()}
          keys={[useModifierKey(), '↵']}
        />
      </div>
    </Modal>,
  );
}
