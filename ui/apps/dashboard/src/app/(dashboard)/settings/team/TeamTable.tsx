'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import Button from '@/components/Button';
import Modal from '@/components/Modal';
import Table from '@/components/Table';
import { graphql } from '@/gql';
import { relativeTime } from '@/utils/date';

const mutation = graphql(`
  mutation DeleteUser($id: ID!) {
    deleteUser(id: $id)
  }
`);

type Props = {
  loggedInUserID: string;
  users: User[];
};

export default function TeamTable({ loggedInUserID, users }: Props) {
  const router = useRouter();
  const [, deleteUserMutation] = useMutation(mutation);
  const [pendingUser, setPendingUser] = useState<User | undefined>(undefined);

  function cancelDeletion() {
    setPendingUser(undefined);
  }

  async function onClickDelete(user: User) {
    setPendingUser(user);
  }

  async function deleteUser(id: string) {
    const res = await deleteUserMutation({ id });
    setPendingUser(undefined);

    if (res.error) {
      toast.error('Failed to delete user');
      return;
    }

    router.refresh();
    toast.success('User deleted');
  }

  const columns = createColumns({
    isDeletionPending: pendingUser !== undefined,
    loggedInUserID,
    onClickDelete,
  });

  return (
    <>
      <div className="w-full max-w-6xl">
        <Table columns={columns} data={users} empty="No users found" />
      </div>

      {pendingUser && (
        <Modal
          className="flex min-w-[600px] max-w-xl flex-col gap-4"
          isOpen={Boolean(pendingUser)}
          onClose={cancelDeletion}
        >
          <p>Are you sure you want to delete {pendingUser.email}?</p>

          <div className="flex flex-row justify-end gap-4">
            <Button onClick={cancelDeletion} variant="secondary">
              No
            </Button>
            <Button onClick={() => deleteUser(pendingUser.id)}>Yes</Button>
          </div>
        </Modal>
      )}
    </>
  );
}

function createColumns({
  isDeletionPending,
  loggedInUserID,
  onClickDelete,
}: {
  isDeletionPending: boolean;
  loggedInUserID: string;
  onClickDelete: (user: User) => void;
}) {
  return [
    { key: 'name', label: 'User' },
    { key: 'email', label: 'Email' },
    {
      key: 'lastLoginAt',
      label: 'Last Seen',
      render: (user: User) => {
        if (typeof user.lastLoginAt === 'string') {
          return relativeTime(user.lastLoginAt);
        }
      },
    },
    {
      key: 'createdAt',
      label: 'Added',
      render: (user: User) => {
        if (typeof user.createdAt === 'string') {
          return dayjs(user.createdAt).format('MMM D, YYYY');
        }
      },
    },
    {
      key: 'deleteUser',
      className: 'w-28',
      render: (user: User) => {
        const isDisabled = isDeletionPending || user.id === loggedInUserID;

        return (
          <Button disabled={isDisabled} onClick={() => onClickDelete(user)} variant="danger">
            Delete user
          </Button>
        );
      },
    },
  ];
}

type User = {
  createdAt: string;
  email: string;
  id: string;
  lastLoginAt?: string | null;
  name?: string | null;
};
