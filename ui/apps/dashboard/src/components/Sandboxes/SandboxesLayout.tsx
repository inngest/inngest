import { useState, type ReactNode } from 'react';
import { Alert } from '@inngest/components/Alert';
import {
  HelperPanelControl,
  HelperPanelFrame,
} from '@inngest/components/HelperPanelControl';
import { AlertModal } from '@inngest/components/Modal';
import { cn } from '@inngest/components/utils/classNames';
import { useOrganization } from '@clerk/tanstack-react-start';
import { RiKey2Line } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import LoadingIcon from '@/components/Icons/LoadingIcon';
import {
  SandboxSecretsPanel,
  type SecretsView,
} from './Secrets/SandboxSecretsPanel';

export function SandboxesLayout({ children }: { children: ReactNode }) {
  const environment = useEnvironment();
  const { membership, isLoaded } = useOrganization();
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<SecretsView>({ kind: 'list' });
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [discardAction, setDiscardAction] = useState<'close' | 'list'>();

  function navigate(action: 'close' | 'list') {
    setDiscardAction(undefined);
    setDirty(false);
    setSaving(false);
    setView({ kind: 'list' });
    if (action === 'close') setOpen(false);
  }

  function requestNavigation(action: 'close' | 'list') {
    if (saving) return;
    if (dirty) setDiscardAction(action);
    else navigate(action);
  }

  return (
    <div className="bg-canvasBase flex min-h-0 flex-1 overflow-hidden">
      <div className={cn('min-w-0 flex-1', open && 'hidden lg:block')}>
        {children}
      </div>
      {open && (
        <aside
          aria-label="Sandbox secrets"
          className="border-subtle min-w-0 flex-1 overflow-hidden border-l lg:w-[440px] lg:flex-none"
        >
          <HelperPanelFrame
            title="Secrets"
            icon={<RiKey2Line className="text-subtle h-5 w-5" />}
            onClose={() => requestNavigation('close')}
          >
            <div className="border-subtle text-muted border-b px-4 py-3 text-xs">
              Environment{' '}
              <span className="text-basis ml-1 font-medium">
                {environment.name}
              </span>
            </div>
            {!isLoaded ? (
              <div className="flex justify-center p-8">
                <LoadingIcon />
              </div>
            ) : membership?.role !== 'org:admin' ? (
              <div className="p-4">
                <Alert severity="info">
                  Only organization admins can view and manage secrets.
                </Alert>
              </div>
            ) : (
              <SandboxSecretsPanel
                environmentID={environment.id}
                view={view}
                onViewChange={setView}
                onBack={() => requestNavigation('list')}
                onSaved={() => navigate('list')}
                onDirtyChange={setDirty}
                onSavingChange={setSaving}
              />
            )}
          </HelperPanelFrame>
        </aside>
      )}
      <div className="shrink-0">
        <HelperPanelControl
          activeTitle={open ? 'Secrets' : null}
          items={[
            {
              title: 'Secrets',
              icon: <RiKey2Line className="h-5 w-5" />,
              action: () => (open ? requestNavigation('close') : setOpen(true)),
            },
          ]}
        />
      </div>
      <AlertModal
        isOpen={Boolean(discardAction)}
        title="Discard Unsaved Changes?"
        description="Your unsaved secret names and values will be cleared."
        confirmButtonLabel="Discard Changes"
        cancelButtonLabel="Keep Editing"
        confirmButtonKind="danger"
        onClose={() => setDiscardAction(undefined)}
        onSubmit={() => {
          if (discardAction) navigate(discardAction);
        }}
        className="w-full max-w-md"
      />
    </div>
  );
}
