type BaseMessagePart = {
  type: 'text';
  content: string;
};

type ToolCallMessagePart = {
  type: 'tool-call';
  action: () => void;
  content: string;
};

export type MessagePart = BaseMessagePart | ToolCallMessagePart;

export type ConversationMessage = {
  parts: MessagePart[];
  role: 'agent' | 'user';
};
