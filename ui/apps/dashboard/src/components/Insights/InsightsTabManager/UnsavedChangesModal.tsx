import { Alert } from "@inngest/components/Alert/NewAlert";
import { Button } from "@inngest/components/Button/NewButton";
import { Modal } from "@inngest/components/Modal/Modal";

import type { Tab } from "@/components/Insights/types";

interface UnsavedChangesModalProps {
  isOpen: boolean;
  unsavedTabs: Tab[];
  onCancel: () => void;
  onDiscardAll: () => void;
  onSaveAll: () => void;
}

export function UnsavedChangesModal({
  isOpen,
  unsavedTabs,
  onCancel,
  onDiscardAll,
  onSaveAll,
}: UnsavedChangesModalProps) {
  const isSingleTab = unsavedTabs.length === 1;
  const discardLabel = isSingleTab ? "Discard" : "Discard All";
  const saveLabel = isSingleTab ? "Save" : "Save All";

  return (
    <Modal className="w-[656px]" isOpen={isOpen} onClose={onCancel}>
      <Modal.Header>Unsaved changes</Modal.Header>
      <Modal.Body>
        {isSingleTab ? (
          <p className="text-subtle text-sm">
            Unsaved changes: <strong>{unsavedTabs[0]?.name}</strong>
          </p>
        ) : (
          <p className="text-subtle text-sm">
            You have {unsavedTabs.length} unsaved tabs
          </p>
        )}
        <Alert className="mt-4 text-sm" severity="warning">
          Changes in these tabs will be lost if you discard them without saving.
        </Alert>
      </Modal.Body>
      <Modal.Footer>
        <div className="flex justify-end gap-2">
          <Button
            appearance="outlined"
            kind="secondary"
            label="Cancel"
            onClick={onCancel}
          />
          <Button
            appearance="outlined"
            kind="danger"
            label={discardLabel}
            onClick={onDiscardAll}
          />
          <Button kind="primary" label={saveLabel} onClick={onSaveAll} />
        </div>
      </Modal.Footer>
    </Modal>
  );
}
