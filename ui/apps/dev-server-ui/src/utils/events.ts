export const genericiseEvent = (eventDataStr: string | null | undefined) => {
  const data = JSON.parse(eventDataStr ?? '{}');

  return JSON.stringify(
    {
      name: data.name,
      data: data.data,
      user: data.user,
    },
    null,
    2
  );
};
