const payloadContent = `{
  name: 'Function Tabs',
  data: {
    // Each event will always have a "data" object, super long line designed to break outside of its parent container
    // this can be a few fields
    // or a few hundred fields
    fields: { can: 'be nested objects' },
  },
  // optional  
 }`;

const schemaContent = `{
  label: Function Schema,
  id: ObjectId,
  count: Integer,
 }`;

export const funcTabs = [
  {
    label: 'Payload',
    content: payloadContent,
  },
  {
    label: 'Schema',
    content: schemaContent,
  },
];
