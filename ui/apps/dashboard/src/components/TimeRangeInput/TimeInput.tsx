'use client';

import { useReducer, useRef } from 'react';
import * as Popover from '@radix-ui/react-popover';
import * as chrono from 'chrono-node';
import { useDebounce } from 'react-use';

import Input from '@/components/Forms/Input';

type Props = {
  onChange: (newDateTime: Date) => void;
  placeholder?: string;
  required?: boolean;
};

type State =
  | {
      inputString: string;
      suggestedDateTime: Date | undefined;
      status: 'typing' | 'idle' | 'focused';
    }
  | {
      inputString: string;
      suggestedDateTime: Date;
      status: 'suggestion_available';
    }
  | {
      inputString: string;
      suggestedDateTime: Date | undefined;
      status: 'suggestion_applied' | 'pasted_date_applied';
    };

type Action =
  | {
      type: 'focused';
    }
  | {
      type: 'applied_example';
      exampleDate: Date;
    }
  | {
      type: 'blurred';
    }
  | {
      type: 'typed';
      value: string;
    }
  | {
      type: 'pasted_date';
      pastedText: string;
      pastedDate: Date;
    }
  | {
      type: 'stopped_typing';
    }
  | {
      type: 'applied_suggestion';
    };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'focused':
      return {
        ...state,
        status: 'focused',
      };
    case 'applied_example':
      return {
        inputString: action.exampleDate.toLocaleString(),
        suggestedDateTime: action.exampleDate,
        status: 'suggestion_applied',
      };
    case 'blurred':
      return {
        ...state,
        status: 'idle',
      };
    case 'typed':
      return {
        ...state,
        inputString: action.value,
        suggestedDateTime: undefined,
        status: 'typing',
      };
    case 'pasted_date':
      return {
        ...state,
        inputString: action.pastedText,
        status: 'pasted_date_applied',
      };
    case 'stopped_typing':
      // Chrono's types are wrong - parseData can return undefined
      const parsedDateTime = chrono.parseDate(state.inputString) as unknown as Date | null;

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

export function TimeInput({ onChange, placeholder, required }: Props) {
  const inputRef = useRef<HTMLInputElement>(null);
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
      if (state.status === 'suggestion_applied' || state.status === 'pasted_date_applied') return;
      dispatch({ type: 'stopped_typing' });
      onChange(state.suggestedDateTime);
    },
    350,
    [state.inputString]
  );

  function handleInputFocus() {
    dispatch({ type: 'focused' });
  }

  function handleInputBlur(event: React.FocusEvent<HTMLInputElement>) {
    // If we click on the popover, we don't want to blur the input.
    if (event.relatedTarget?.closest('[data-radix-popper-content-wrapper]')) return;
    dispatch({ type: 'blurred' });
  }

  function handleExampleClick(example: string) {
    const exampleDate = chrono.parseDate(example);
    // Focus the input after applying the example so that the user can tab to the next element.
    inputRef.current?.focus();

    dispatch({ type: 'applied_example', exampleDate });
    onChange(exampleDate);
  }

  function handleInputChange(event: React.ChangeEvent<HTMLInputElement>) {
    // Check if this is a paste event. If so, we don't want to dispatch the typed action.
    if (event.nativeEvent.inputType === 'insertFromPaste') return;
    dispatch({ type: 'typed', value: event.target.value });
  }

  function handleInputPaste(event: React.ClipboardEvent<HTMLInputElement>) {
    const pastedText = event.clipboardData.getData('text');
    const pastedDate = chrono.parseDate(pastedText);
    // Chrono's types are wrong - parseData can return null.
    // See https://github.com/wanasit/chrono/pull/544
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (!pastedDate) {
      dispatch({ type: 'typed', value: pastedText });
      return;
    }
    dispatch({ type: 'pasted_date', pastedDate, pastedText });
    onChange(pastedDate);
  }

  function handleInputKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (!(event.code === 'Enter' && state.status === 'suggestion_available')) {
      return;
    }
    event.preventDefault();
    dispatch({ type: 'applied_suggestion' });
    onChange(state.suggestedDateTime);
  }

  function handleSuggestionClick() {
    if (state.status !== 'suggestion_available') return;
    dispatch({ type: 'applied_suggestion' });
    onChange(state.suggestedDateTime);
  }

  return (
    <Popover.Root open={state.status === 'focused' || state.status === 'suggestion_available'}>
      <Popover.Anchor>
        <Input
          type="text"
          value={state.inputString}
          placeholder={placeholder}
          onChange={handleInputChange}
          onPaste={handleInputPaste}
          onKeyDown={handleInputKeyDown}
          onFocus={handleInputFocus}
          onBlur={handleInputBlur}
          required={required}
          ref={inputRef}
        />
      </Popover.Anchor>
      <Popover.Portal>
        <>
          {state.status === 'focused' && (
            <Popover.Content
              className="shadow-floating z-[100] w-[--radix-popover-trigger-width] space-y-2 rounded-md bg-white/95 p-2 text-sm ring-1 ring-black/5 backdrop-blur-[3px]"
              sideOffset={5}
              onOpenAutoFocus={(event) => event.preventDefault()}
            >
              <p className="text-slate-500">Type a date and/or time</p>
              <div className="flex flex-wrap gap-1 text-black">
                {[
                  '10 AM',
                  '1 hour ago',
                  'yesterday',
                  '3 days ago',
                  '15:30:24',
                  'Jan 14',
                  '2024-01-01T10:00:00Z',
                ].map((example) => (
                  <button
                    className="h-5 rounded bg-slate-200 px-1.5 text-xs text-black hover:bg-slate-300"
                    type="button"
                    key={example}
                    onClick={() => handleExampleClick(example)}
                  >
                    {example}
                  </button>
                ))}
              </div>
            </Popover.Content>
          )}
          {state.status === 'suggestion_available' && (
            <Popover.Content
              className="shadow-floating z-[100] inline-flex cursor-pointer items-center gap-2 rounded-md bg-white/95 p-2 text-sm text-slate-800 ring-1 ring-black/5 backdrop-blur-[3px]"
              sideOffset={5}
              onOpenAutoFocus={(event) => event.preventDefault()}
              onClick={handleSuggestionClick}
            >
              {state.suggestedDateTime.toLocaleString()}
              <kbd
                className="ml-auto flex h-6 w-6 items-center justify-center rounded bg-slate-100 p-2 font-sans text-xs"
                aria-label="Press Enter to apply the parsed date and time."
              >
                ↵
              </kbd>
            </Popover.Content>
          )}
        </>
      </Popover.Portal>
    </Popover.Root>
  );
}
