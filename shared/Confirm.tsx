import React from "react";
import Modal from "./Modal";
import Button, { Kinds } from "src/shared/Button";
import styled from "@emotion/styled";

export type Confirm = {
  // prompt is the text to show on confirm
  prompt: string;
  detail?: string;

  // confirm button text
  confirm?: string;
  kind?: Kinds;
  // cancel button text
  cancel?: string;

  onConfirm?: () => void;
  onClose?: () => void;
  onCancel?: () => void;
};

export type Context = {
  newConfirm: (c: Confirm) => Promise<void>;
};

const noop = (_: Confirm): Promise<void> => {
  return new Promise((resolve) => resolve());
};

const ConfirmContext = React.createContext<Context>({
  newConfirm: noop,
});

// Confirm renders the confirm component in the root of our app.
export const ConfirmWrapper: React.FC<{}> = (props) => {
  const resolve = React.useRef<() => void | null>(null);
  const reject = React.useRef<() => void | null>(null);

  const [confirm, setConfirm] = React.useState<Confirm | null>(null);

  const newConfirm = React.useCallback(
    (c: Confirm): Promise<void> => {
      if (confirm) {
        console.warn("already showing confirm, can't add new");
        return noop(c);
      }

      const p = new Promise<void>((res, rej) => {
        (resolve.current as any) = () => {
          res();
          setConfirm(null);
        };
        (reject.current as any) = () => {
          rej();
          setConfirm(null);
        };
      });
      setConfirm(c);
      return p;
    },
    [setConfirm]
  );

  return (
    <ConfirmContext.Provider value={{ newConfirm }}>
      {props.children}
      {confirm && (
        <Modal
          onClose={() => {
            setConfirm(null);
          }}
        >
          <Wrapper>
            <div>
              <p>
                <b>{confirm.prompt}</b>
              </p>
            </div>
            <div style={{ padding: 20 }}>
              <Button onClick={() => reject.current && reject.current()}>
                {confirm.cancel || "No"}
              </Button>
              <Button
                kind={confirm.kind || "danger"}
                onClick={() => resolve.current && resolve.current()}
              >
                {confirm.confirm || "Yes"}
              </Button>
            </div>
          </Wrapper>
        </Modal>
      )}
    </ConfirmContext.Provider>
  );
};

export const ConfirmModal: React.FC<Confirm> = ({
  onClose,
  onCancel,
  onConfirm,
  prompt,
  detail,
  confirm,
  kind,
  cancel,
}) => {
  return (
    <Modal onClose={onClose ? onClose : () => {}}>
      <Wrapper>
        <div>
          <p>
            <b>{prompt}</b>
          </p>
          {detail && <p style={{ marginTop: 15, opacity: 0.8 }}>{detail}</p>}
        </div>
        <div style={{ padding: 20 }}>
          <Button onClick={onCancel ? onCancel : onClose}>
            {cancel || "No"}
          </Button>
          <Button kind={kind || "danger"} onClick={onConfirm}>
            {confirm || "Yes"}
          </Button>
        </div>
      </Wrapper>
    </Modal>
  );
};

export const useConfirm = () => {
  return React.useContext(ConfirmContext);
};

export default useConfirm;

const Wrapper = styled.div`
  display: block;
  background: #fff;
  min-width: 400px;
  max-width: 500px;
  text-align: center;
  border-radius: 3px;

  > div:first-of-type {
    padding: 40px;
    border-bottom: 1px solid #eee;
  }
`;
