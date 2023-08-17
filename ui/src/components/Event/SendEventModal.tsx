import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import Editor, { useMonaco } from '@monaco-editor/react';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import Modal from '@/components/Modal';
import { usePortal } from '../../hooks/usePortal';
import { useSendEventMutation } from '../../store/devApi';
import { selectEvent } from '../../store/global';
import { useAppDispatch } from '../../store/hooks';
import { genericiseEvent } from '../../utils/events';

export default function SendEventModal({ data, isOpen, onClose }) {
  const [_sendEvent, sendEventState] = useSendEventMutation();
  const portal = usePortal();
  const eventDataStr = data;
  const dispatch = useAppDispatch();

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
        dispatch(selectEvent(data.id));
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
      rules: [],
      colors: {
        'editor.background': '#1e293b', // slate-800
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
      description="Send an event manually by pasting a payload or creating a new one"
      className="max-w-5xl w-full"
    >
      <div className="m-4">
        <div className="relative w-full h-[30rem] flex flex-col rounded overflow-hidden">
          <div className="mt-4 items-center bg-slate-800 shadow border-b border-slate-700/20 flex justify-between rounded-t">
            <p className=" text-slate-300/50 text-xs px-5">Payload</p>
            <div className="flex gap-2 items-center mr-2">
              <div className="py-2 flex flex-row items-center space-x-2">
                <div className="text-4xs text-center text-white">Cmd+Enter</div>
                <button
                  onClick={() => sendEvent()}
                  className="bg-slate-700/50 hover:bg-slate-700/80 border-slate-700/50 flex gap-1.5 items-center border text-xs rounded-sm px-2.5 py-1 text-slate-100 transition-all duration-150"
                >
                  {sendEventState.isLoading ? 'Spinner' : 'Send event'}
                </button>
              </div>
            </div>
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
                fixedOverflowWidgets: false,
                formatOnPaste: false,
                formatOnType: false,
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
                fontFamily: 'Source Code Pro, monospace',
                fontSize: 13,
                lineHeight: 26,
              }}
            />
          ) : null}
        </div>
      </div>
    </Modal>,
  );
}
