import { useEffect, useState } from 'react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@inngest/components/DropdownMenu/DropdownMenu';

import { KeyboardShortcut } from '../KeyboardShortcut';

type SQLEditorContextMenuProps = {
  onCopy: () => void;
  onCut: () => void;
  onPaste: () => void;
  onPrettifySQL: () => void;
  onRunQuery: () => void;
  onSaveQuery: () => void;
  hasSelection: () => boolean;
  hasUnsavedChanges: boolean;
};

export function SQLEditorContextMenu({
  onCopy,
  onCut,
  onPaste,
  onPrettifySQL,
  onRunQuery,
  onSaveQuery,
  hasSelection,
  hasUnsavedChanges,
}: SQLEditorContextMenuProps) {
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    hasSelection: boolean;
  } | null>(null);

  useEffect(() => {
    const handleContextMenu = (e: MouseEvent) => {
      // Check if the event target is within the Monaco editor
      const target = e.target as HTMLElement;
      const editorContainer = target.closest('.monaco-editor');

      if (editorContainer) {
        e.preventDefault();
        setContextMenu({
          x: e.clientX,
          y: e.clientY,
          hasSelection: hasSelection(),
        });
      }
    };

    const handleClick = () => {
      setContextMenu(null);
    };

    document.addEventListener('contextmenu', handleContextMenu);
    document.addEventListener('click', handleClick);

    return () => {
      document.removeEventListener('contextmenu', handleContextMenu);
      document.removeEventListener('click', handleClick);
    };
  }, [hasSelection]);

  if (!contextMenu) return null;

  return (
    <DropdownMenu
      open={!!contextMenu}
      onOpenChange={(open) => !open && setContextMenu(null)}
    >
      <DropdownMenuContent
        align="start"
        style={{
          position: 'fixed',
          left: `${contextMenu.x}px`,
          top: `${contextMenu.y}px`,
        }}
      >
        <DropdownMenuItem onSelect={onPrettifySQL}>
          <span>Format SQL</span>
          <span className="ml-auto">
            <KeyboardShortcut
              className="py-0"
              color="text-muted"
              keys={['shift', 'alt', 'F']}
            />
          </span>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem disabled={!contextMenu.hasSelection} onSelect={onCut}>
          <span>Cut</span>
          <span className="ml-auto">
            <KeyboardShortcut
              className="py-0"
              color="text-muted"
              keys={['cmd', 'ctrl', 'X']}
            />
          </span>
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={!contextMenu.hasSelection}
          onSelect={onCopy}
        >
          <span>Copy</span>
          <span className="ml-auto">
            <KeyboardShortcut
              className="py-0"
              color="text-muted"
              keys={['cmd', 'ctrl', 'C']}
            />
          </span>
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={onPaste}>
          <span>Paste</span>
          <span className="ml-auto">
            <KeyboardShortcut
              className="py-0"
              color="text-muted"
              keys={['cmd', 'ctrl', 'V']}
            />
          </span>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={onRunQuery}>
          <span>Run query</span>
          <span className="ml-auto">
            <KeyboardShortcut
              className="py-0"
              color="text-muted"
              keys={['cmd', 'ctrl', 'enter']}
            />
          </span>
        </DropdownMenuItem>
        <DropdownMenuItem disabled={!hasUnsavedChanges} onSelect={onSaveQuery}>
          <span>Save query</span>
          <span className="ml-auto">
            <KeyboardShortcut
              className="py-0"
              color="text-muted"
              keys={['cmd', 'ctrl', 'alt', 'S']}
            />
          </span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
