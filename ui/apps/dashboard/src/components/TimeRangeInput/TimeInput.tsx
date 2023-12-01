'use client';

import { useReducer } from 'react';
import * as Popover from '@radix-ui/react-popover';
import * as chrono from 'chrono-node';
import { useDebounce } from 'react-use';

import Input from '@/components/Forms/Input';

type Props = {
  onChange: (newDateTime: Date) => void;
  required?: boolean;
};

type State =
  | {
      inputString: string;
      suggestedDateTime: undefined;
      status: 'typing' | 'idle';
    }
  | {
      inputString: string;
      suggestedDateTime: Date;
      status: 'suggestion_available';
    }
  | {
      inputString: string;
      suggestedDateTime: Date | undefined;
      status: 'suggestion_applied';
    };

type Action =
  | {
      type: 'typed_string';
      string: string;
    }
  | {
      type: 'stopped_typing';
    }
  | {
      type: 'applied_suggestion';
    };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'typed_string':
      return {
        ...state,
        inputString: action.string,
        suggestedDateTime: undefined,
        status: 'typing',
      };
    case 'stopped_typing':
      const parsedDateTime = chrono.parseDate(state.inputString);
      if (!parsedDateTime) {
        return {
          ...state,
          suggestedDateTime: undefined,
          status: 'idle',
        };
      }
      return {
        ...state,
        suggestedDateTime: parsedDateTime,
        status: 'suggestion_available',
      };
    case 'applied_suggestion':
      return {
        ...state,
        inputString: state.suggestedDateTime?.toLocaleString() ?? '',
        status: 'suggestion_applied',
      };
    default:
      throw new Error('Unknown action type');
  }
}

export function TimeInput({ onChange, required }: Props) {
  const [state, dispatch] = useReducer(reducer, {
    inputString: '',
    suggestedDateTime: undefined,
    status: 'idle',
  });
  useDebounce(
    () => {
      // Debounce gets called when a suggestion is applied, which doesn't mean the user stopped
      // typing, so we need to check if the suggestion was applied before we dispatch the
      // stopped_typing action.
      if (state.status === 'suggestion_applied') return;
      dispatch({ type: 'stopped_typing' });
    },
    350,
    [state.inputString]
  );

  function onInputChange(event: React.ChangeEvent<HTMLInputElement>) {
    dispatch({ type: 'typed_string', string: event.target.value });
  }

  function onInputKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.code === 'Enter' && state.status === 'suggestion_available') {
      event.preventDefault();
      dispatch({ type: 'applied_suggestion' });
      onChange(state.suggestedDateTime);
    }
  }

  return (
    <Popover.Root open={state.status === 'suggestion_available'}>
      <Popover.Anchor>
        <Input
          type="text"
          value={state.inputString}
          onChange={onInputChange}
          onKeyDown={onInputKeyDown}
          required={required}
        />
      </Popover.Anchor>
      <Popover.Portal>
        <Popover.Content
          className="shadow-floating z-[100] inline-flex items-center gap-2 space-y-4 rounded-md bg-white/95 p-2 text-sm text-slate-800 ring-1 ring-black/5 backdrop-blur-[3px]"
          sideOffset={5}
          onOpenAutoFocus={(event) => event.preventDefault()}
        >
          {state.suggestedDateTime?.toLocaleString()}
          <kbd
            className="ml-auto flex h-6 w-6 items-center justify-center rounded bg-slate-100 p-2 font-sans text-xs"
            aria-label="Press Enter to apply the parsed date and time."
          >
            ↵
          </kbd>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
