'use client';

import { useState } from 'react';
import { notFound } from 'next/navigation';
import { PencilIcon, TrashIcon } from '@heroicons/react/20/solid';
import { CodeKey } from '@inngest/components/CodeKey';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { RiMore2Line } from '@remixicon/react';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import { Provider } from './Context';
import DeleteKeyModal from './DeleteKeyModal';
import EditKeyNameModal from './EditKeyNameModal';
import FilterEvents from './FilterEvents';
import TransformEvent from './TransformEvent';

const GetKeyDocument = graphql(`
  query GetIngestKey($environmentID: ID!, $keyID: ID!) {
    environment: workspace(id: $environmentID) {
      ingestKey(id: $keyID) {
        id
        name
        createdAt
        presharedKey
        url
        filter {
          type
          ips
          events
        }
        metadata
        source
      }
    }
  }
`);

type KeyDetailsProps = {
  params: {
    environmentSlug: string;
    ingestKeys: string;
    keyID: string;
  };
};

const SOURCE_INTEGRATION = 'integration';

export default function Keys({ params: { ingestKeys, keyID } }: KeyDetailsProps) {
  const [isDeleteKeyModalVisible, setIsDeleteKeyModalVisible] = useState(false);
  const [isEditKeyNameModalVisible, setIsEditKeyNameModalVisible] = useState(false);

  const environment = useEnvironment();

  const { data, isLoading, error } = useGraphQLQuery({
    query: GetKeyDocument,
    variables: {
      environmentID: environment.id,
      keyID,
    },
  });

  if (isLoading) {
    return <>{/* To do: skeleton */}</>;
  }

  const key = data?.environment.ingestKey;

  if (error || !key) {
    notFound();
  }

  const filterType = key.filter.type;
  if (!filterType || !isFilterType(filterType)) {
    throw new Error(`invalid filter type: ${filterType}`);
  }

  // Integration created keys cannot be deleted or renamed
  const isIntegration = key.source === SOURCE_INTEGRATION;

  let value = '',
    maskedValue = '',
    keyLabel = 'Key';
  if (ingestKeys === 'webhooks') {
    value = key.url || '';
    // Leave the base url + the beginning of the key
    maskedValue = value.replace(/(.{0,}\/e\/)(\w{0,8}).+/, '$1$2');
    keyLabel = 'Webhook URL';
  } else {
    value = key.presharedKey;
    maskedValue = value.substring(0, 8);
    keyLabel = 'Event Key';
  }

  return (
    <div className="m-6 divide-y divide-slate-100">
      <Provider initialState={key}>
        <div className="pb-8">
          <div className="mb-8 flex items-center gap-1">
            <h2 className="text-lg font-semibold">{key.name}</h2>
            <DropdownMenu>
              <DropdownMenuTrigger className="relative data-[state=open]:before:absolute data-[state=open]:before:-bottom-3 data-[state=open]:before:left-0 data-[state=open]:before:h-6 data-[state=open]:before:w-6 data-[state=open]:before:rounded-full data-[state=open]:before:bg-slate-100">
                <RiMore2Line className="absolute -top-2 left-1 z-10 h-4 w-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem onSelect={() => setIsEditKeyNameModalVisible(true)}>
                  <PencilIcon className="h-4 w-4" />
                  Edit Name
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => setIsDeleteKeyModalVisible(true)}>
                  <TrashIcon className="h-4 w-4" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <DeleteKeyModal
              keyID={keyID}
              isOpen={isDeleteKeyModalVisible}
              onClose={() => setIsDeleteKeyModalVisible(false)}
              description={
                isIntegration
                  ? 'Warning: This key was created via integration. Please confirm that you are no longer using it before deleting.'
                  : ''
              }
            />
            <EditKeyNameModal
              keyID={keyID}
              keyName={key.name}
              isOpen={isEditKeyNameModalVisible}
              onClose={() => setIsEditKeyNameModalVisible(false)}
            />
            {key.source === SOURCE_INTEGRATION && (
              <span className="ml-8 text-sm text-slate-400">Created via integration</span>
            )}
          </div>
          <div className="w-3/5">
            <CodeKey fullKey={value} maskedKey={maskedValue} label={keyLabel} />
          </div>
        </div>
        <TransformEvent keyID={keyID} metadata={key.metadata} keyName={key.name} />
        <FilterEvents
          keyID={keyID}
          filter={{
            ...key.filter,
            type: filterType,
          }}
          keyName={key.name}
        />
      </Provider>
    </div>
  );
}

function isFilterType(value: string): value is 'allow' | 'deny' {
  return value === 'allow' || value === 'deny';
}
