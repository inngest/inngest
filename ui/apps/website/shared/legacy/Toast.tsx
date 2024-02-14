// @ts-nocheck
import React, { useEffect, useState, useReducer, useContext } from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";
import uuid from "src/utils/uuid";

export enum ToastTypes {
  success,
  error,
  default,
}

export type ToastTypeStrings = keyof typeof ToastTypes;

export type Toast = {
  id?: string; // ID to dedupe
  type: ToastTypeStrings;
  message: string;
  sticky?: boolean;
  icon?: any;
  // duration in seconds
  duration?: number;
};

export type Context = {
  push: (t: Toast) => () => void;
  remove: (id: string) => void;
};

const ToastContext = React.createContext<Context>({
  push: (t: Toast) => () => {},
  remove: (id: String) => {},
});

type InternalToast = Toast & { id: string };

type Action =
  | { type: "add"; toast: InternalToast }
  | { type: "remove"; id: string };

type State = InternalToast[];

const reducer = (s: State, a: Action) => {
  switch (a.type) {
    case "add":
      // don't duplicate if ID exists
      if (s.find((s) => s.id === a.toast.id)) {
        return s;
      }
      return s.concat([{ ...a.toast }]);
    case "remove":
      return s.filter((t) => t.id !== a.id);
  }
  return s;
};

// ToastWrapper is a top level component that renders all toasts to
// the UI and manages their lifecycles.
export const ToastWrapper = ({ children }: { children: React.ReactNode }) => {
  const [state, dispatch] = useReducer(reducer, []);

  const remove = (id: string) => dispatch({ type: "remove", id });
  const push = (t: Toast) => {
    const id = uuid();
    dispatch({ type: "add", toast: { ...t, id: t.id || id } });
    return () => remove(id);
  };

  return (
    <ToastContext.Provider value={{ push, remove }}>
      <Wrapper>
        {state.map((t, n) => (
          <ToastItem key={t.id} toast={t} />
        ))}
      </Wrapper>
      {children}
    </ToastContext.Provider>
  );
};

export const useToast = () => {
  return useContext(ToastContext);
};

export default useToast;

const Wrapper = styled.div`
  position: fixed;
  top: 20px;
  width: 100vw;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  z-index: 1001;
  pointer-events: none;
`;
const Item = styled.div`
  text-align: center;
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.15), 0 0 5px rgba(0, 0, 0, 0.15);
  border-radius: 4px;
  padding: 8px 16px;
  font-size: 0.9rem;
  display: flex;
  align-items: center;

  transition: all 0.5s;
  opacity: 0;
  transform: translateY(-10px);

  margin-bottom: 10px;
  max-width: 300px;
  background: #fff;
`;

const success = css`
  background: #23a282;
  color: #fff;
`;

const error = css`
  background: #cb2525;
  color: #fff;
`;

const visible = css`
  opacity: 1;
  transform: translateY(0);
`;

const defaultDuration = 5000;

const ToastItem = ({ toast }: { toast: InternalToast }) => {
  const { remove } = useToast();
  const [shown, setShown] = useState(false);

  const I = toast.icon;

  useEffect(() => {
    // Fade in nicely.
    setTimeout(() => {
      setShown(true);
    }, 25);

    // And hide this toast after N seconds
    setTimeout(() => {
      !toast.sticky && setShown(false);
    }, toast.duration || defaultDuration - 500);
    setTimeout(() => {
      !toast.sticky && remove(toast.id);
    }, toast.duration || defaultDuration);
  }, [toast.id]);

  return (
    <Item
      onClick={() => remove(toast.id)}
      css={[
        toast.type === "success" && success,
        toast.type === "error" && error,
        shown && visible,
      ]}
    >
      {I && <I size={22} style={{ marginRight: 10 }} />}
      <span>
        {toast.message.replace("[GraphQL] ", "").replace("[Network] ", "")}
      </span>
    </Item>
  );
};
