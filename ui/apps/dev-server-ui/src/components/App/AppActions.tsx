import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { AlertModal } from '@inngest/components/Modal';
import { RiDeleteBinLine, RiMore2Line } from '@remixicon/react';
import { toast } from 'sonner';

import { useDeleteAppMutation } from '@/store/generated';

export default function AppActions({ id, name }: { id: string; name: string }) {
  const [isAlertModalOpen, setIsAlertModalOpen] = useState(false);
  const [_deleteApp] = useDeleteAppMutation();

  async function deleteApp() {
    try {
      const response = await _deleteApp({
        id: id,
      });
      toast.success(`${name || 'The app'} was successfully deleted.`);
      console.log('Deleted app:', response);
    } catch (error) {
      toast.error(`${name || 'The app'} could not be deleted: ${error}`);
      console.error('Error deleting app:', error);
    }
    // To do: add optimistic render in the list
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button kind="secondary" appearance="outlined" size="medium" icon={<RiMore2Line />} />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem className="text-error" onSelect={() => setIsAlertModalOpen(true)}>
            <RiDeleteBinLine className="h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <AlertModal
        className="w-1/3 max-w-xl"
        isOpen={isAlertModalOpen}
        title="Are you sure you want to delete the app?"
        onClose={() => setIsAlertModalOpen(false)}
        onSubmit={deleteApp}
      />
    </>
  );
}
