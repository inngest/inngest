const payloadContent = `{
  name: 'some.scope/event.name',
  data: {
    // Each event will always have a "data" object, super long line designed to break outside of its parent container
    // this can be a few fields
    // or a few hundred fields
    fields: { can: 'be nested objects' },
  },
  // optional
  user: {
    id: '1234567890',
  },
  ts: 1667221378334, // This will be in every event
  v: '2022-10-31.1', // optional,
  name: 'some.scope/event.name',
  data: {
    // Each event will always have a "data" object, super long line designed to break outside of its parent container
    // this can be a few fields
    // or a few hundred fields
    fields: { can: 'be nested objects' },
  },
  // optional
  user: {
    id: '1234567890',
  },
  ts: 1667221378334, // This will be in every event
  v: '2022-10-31.1', // optional,
  name: 'some.scope/event.name',
  data: {
    // Each event will always have a "data" object, super long line designed to break outside of its parent container
    // this can be a few fields
    // or a few hundred fields
    fields: { can: 'be nested objects' },
  },
  // optional
  user: {
    id: '1234567890',
  },
  ts: 1667221378334, // This will be in every event
  v: '2022-10-31.1', // optional,
  name: 'some.scope/event.name',
  data: {
    // Each event will always have a "data" object, super long line designed to break outside of its parent container
    // this can be a few fields
    // or a few hundred fields
    fields: { can: 'be nested objects' },
  },
  // optional
  user: {
    id: '1234567890',
  },
  ts: 1667221378334, // This will be in every event
  v: '2022-10-31.1', // optional,
  name: 'some.scope/event.name',
  data: {
    // Each event will always have a "data" object, super long line designed to break outside of its parent container
    // this can be a few fields
    // or a few hundred fields
    fields: { can: 'be nested objects' },
  },
  // optional
  user: {
    id: '1234567890',
  },
  ts: 1667221378334, // This will be in every event
  v: '2022-10-31.1', // optional
 }`;

const schemaContent = `{
  label: String,
  id: ObjectId,
  count: Integer,
 }`;

export const eventTabs = [
  {
    label: 'Payload',
    content: payloadContent,
  },
  {
    label: 'Schema',
    content: schemaContent,
  },
];
