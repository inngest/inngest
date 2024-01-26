'use client';

import { createContext, useState } from 'react';
import { useMutation, type CombinedError } from 'urql';

import { graphql } from '@/gql';
import type { GetIngestKeyQuery } from '@/gql/graphql';

type IngestKey = Omit<GetIngestKeyQuery['environment']['ingestKey'], '__typename'>;
type PartialIngestKey = Partial<IngestKey>;

type IngestKeyContext = {
  fetching?: boolean;
  state?: IngestKey;
  save: (s: PartialIngestKey) => Promise<{ error: CombinedError | Error | undefined }>;
};

export const Context = createContext<IngestKeyContext>({
  save: async () => {
    console.log('warning: must use provider');
    return { error: undefined };
  },
});

const UpdateIngestKeyDocument = graphql(`
  mutation UpdateIngestKey($id: ID!, $input: UpdateIngestKey!) {
    updateIngestKey(id: $id, input: $input) {
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
    }
  }
`);

export function Provider({
  initialState,
  children,
}: {
  initialState: IngestKey;
  children: React.ReactNode;
}) {
  const [state, setState] = useState<IngestKey>(initialState);
  const [{ fetching }, updateIngestKey] = useMutation(UpdateIngestKeyDocument);

  async function save(updates: PartialIngestKey) {
    if (updates.id !== state.id) {
      return { error: new Error('ID did not match when saving state') };
    }
    const newState = { ...state, ...updates };
    setState(newState);
    const result = await updateIngestKey({
      id: state.id,
      input: {
        // Use new state to pass all inputs as null fields will unset
        name: newState.name,
        filterList: newState.filter,
        metadata: newState.metadata,
      },
    });
    return { error: result.error };
  }

  return (
    <Context.Provider
      value={{
        state,
        save,
        fetching,
      }}
    >
      {children}
    </Context.Provider>
  );
}
