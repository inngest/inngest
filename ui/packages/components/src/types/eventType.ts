import type React from 'react';

export type EventType = {
  name: string;
  archived: boolean;
  functions: any[];
  volume?: {
    totalVolume: number;
    dailyVolumeSlots: {
      startCount: number;
    }[];
  };
};

export type EventTypesOrderBy = {
  direction: EventTypesOrderByDirection;
  field: EventTypesOrderByField;
};

export enum EventTypesOrderByDirection {
  Asc = 'ASC',
  Desc = 'DESC',
}

export enum EventTypesOrderByField {
  Name = 'NAME',
}

export type PageInfo = {
  endCursor: string | null;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
  startCursor: string | null;
};
