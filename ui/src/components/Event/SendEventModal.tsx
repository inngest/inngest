import Editor, { useMonaco } from "@monaco-editor/react";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "preact/hooks";
import { usePortal } from "../../hooks/usePortal";
import { useSendEventMutation } from "../../store/devApi";
import { genericiseEvent } from "../../utils/events";
import CodeBlockModal from "../CodeBlock/CodeBlockModal";

interface EventPayload {
  name: string;
  data?: Record<string, any>;
  user?: Record<string, any>;
  ts: number;
}

interface SendEventModalProps {
  // visible: boolean;
  eventDataStr: string | null | undefined;
  onClose: () => void;
}

export const SendEventModal = ({
  // visible,
  eventDataStr,
  onClose,
}: SendEventModalProps) => {
  const [_sendEvent, sendEventState] = useSendEventMutation();
  const portal = usePortal();

  const snippedData = useMemo(
    () => genericiseEvent(eventDataStr),
    [eventDataStr]
  );

  const [input, setInput] = useState(snippedData);
  useEffect(() => {
    setInput(genericiseEvent(snippedData));
  }, [eventDataStr /*, visible */]);

  const pushToast = (message: string) => {
    alert(message);
  };

  const sendEvent = useCallback<() => void>(() => {
    let data: any;

    try {
      data = JSON.parse(input || "");

      if (!data.ts || typeof data.ts !== "number") {
        data.ts = Date.now();
      }
    } catch (err) {
      return pushToast("Event payload could not be parsed as JSON.");
    }

    if (!data.name) {
      return pushToast("Event payload must contain a name.");
    }

    if (typeof data.name !== "string") {
      return pushToast(
        "Event payload name must be a string, ideally in the format 'scope/subject.verb'."
      );
    }

    if (data.data && typeof data.data !== "object") {
      return pushToast("Event payload data must be an object if defined.");
    }

    if (data.user && typeof data.user !== "object") {
      return pushToast("Event payload user must be an object if defined.");
    }

    _sendEvent(JSON.stringify(data))
      .unwrap()
      .then(() => {
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

    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
      validate: true,
      schemas: [
        {
          uri: "https://inngest.com/event-schema.json",
          fileMatch: ["*"],
          schema: {
            type: "object",
            properties: {
              name: {
                type: "string",
                description:
                  "A unique identifier for the event. The recommended format is scope/subject.verb, e.g. 'app/user.created' or 'stripe/payment.succeeded'.",
              },
              data: {
                type: "object",
                additionalProperties: true,
                description: "Any data pertinent to the event.",
              },
              user: {
                type: "object",
                additionalProperties: true,
                description:
                  "Any user data associated with the event. All fields ending in '_id' will be used to attribute the event to a particular user.",
              },
              ts: {
                type: "number",
                multipleOf: 1,
                minimum: 0,
                description:
                  "An integer representing the milliseconds since the unix epoch at which this event occured. If omitted, the current time will be used.",
              },
            },
            required: ["name"],
          },
        },
      ],
    });
  }, [monaco]);

  // if (!visible) {
  //   return null;
  // }

  return portal(
    <CodeBlockModal closeModal={onClose}>
      <div className="relative w-[60rem] h-[30rem] bg-slate-800/30 border border-slate-700/30 flex flex-col">
        <div className="mt-4 mx-4 bg-slate-800/40 flex shadow border-b border-slate-700/20 flex justify-between rounded-t">
          <div className="flex -mb-px">
            <button className="border-indigo-400 text-white text-xs px-5 py-2.5 border-b block transition-all duration-150">
              Payload
            </button>
          </div>
          <div className="flex gap-2 items-center mr-2">
            <div className="py-2 flex flex-row items-center space-x-2">
              <div className="text-4xs text-center">Cmd+Enter</div>
              <button
                onClick={() => sendEvent()}
                className="bg-slate-700/50 hover:bg-slate-700/80 border-slate-700/50 flex gap-1.5 items-center border text-xs rounded-sm px-2.5 py-1 text-slate-100 transition-all duration-150"
              >
                {sendEventState.isLoading ? "Spinner" : "Send event"}
              </button>
            </div>
          </div>
        </div>
        {monaco ? (
          <Editor
            defaultLanguage="json"
            value={input ?? "{}"}
            onChange={(value) => setInput(value || "")}
            className="overflow-x-hidden flex-1 mx-4 mb-4"
            theme="vs-dark"
            onMount={(editor) => {
              editor.addAction({
                id: "sendInngestEvent",
                label: "Send Inngest Event",
                keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
                contextMenuGroupId: "2_execution",
                run: () => {
                  sendEventRef.current();
                },
              });
            }}
            options={{
              fixedOverflowWidgets: true,
              formatOnPaste: true,
              formatOnType: true,
              minimap: {
                enabled: false,
              },
              lineNumbers: "off",
              extraEditorClassName: "",
              theme: "vs-dark",
              contextmenu: false,
              inlayHints: {
                enabled: "on",
              },
              scrollBeyondLastLine: false,
              wordWrap: "on",
            }}
          />
        ) : null}
      </div>
    </CodeBlockModal>
  );
};
