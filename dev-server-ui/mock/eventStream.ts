/*
  Statuses
  
  - RUNNING
  - COMPLETED
  - FAILED
  - PAUSED
  - ACTION_REQ
  - NO_FN

  */

const eventStream = [
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'RUNNING',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'FAILED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'PAUSED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'ACTION_REQ',
    name: 'stripe/payment.success',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'NO_FN',
    name: 'accounts/super.long.event.name.that.goes.on.forever',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
];

const fnLog = [
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'RUNNING',
    name: 'Function One',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'FAILED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'PAUSED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'ACTION_REQ',
    name: 'stripe/payment.success',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'NO_FN',
    name: 'accounts/super.long.event.name.that.goes.on.forever',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
  {
    datetime: '2018-01-01T00:00:00Z',
    status: 'COMPLETED',
    name: 'accounts/profile.photo.uploaded',
    badge: 1,
  },
];

export const feeds = [
  {
    name: 'Event Stream',
    content: eventStream,
  },
  {
    name: 'Function Log',
    content: fnLog,
  },
];
