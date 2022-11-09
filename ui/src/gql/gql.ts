/* eslint-disable */
import * as types from './graphql';
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';

const documents = {
    "\n  query GetEventsStream($query: EventsQuery!) {\n    events(query: $query) {\n      id\n      name\n      createdAt\n      payload\n    }\n  }\n": types.GetEventsStreamDocument,
};

export function graphql(source: "\n  query GetEventsStream($query: EventsQuery!) {\n    events(query: $query) {\n      id\n      name\n      createdAt\n      payload\n    }\n  }\n"): (typeof documents)["\n  query GetEventsStream($query: EventsQuery!) {\n    events(query: $query) {\n      id\n      name\n      createdAt\n      payload\n    }\n  }\n"];

export function graphql(source: string): unknown;
export function graphql(source: string) {
  return (documents as any)[source] ?? {};
}

export type DocumentType<TDocumentNode extends DocumentNode<any, any>> = TDocumentNode extends DocumentNode<  infer TType,  any>  ? TType  : never;