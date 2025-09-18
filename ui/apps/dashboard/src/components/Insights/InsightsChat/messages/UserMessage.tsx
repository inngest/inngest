'use client';

export const UserMessage = ({ message }: { message: { content: string } }) => {
  return (
    <div className="group relative flex justify-end">
      <div className="text-text-basis mb-2 inline-block max-w-[340px] whitespace-pre-wrap rounded-md bg-gray-100 px-3 py-2 text-sm dark:bg-[#353535]">
        {message.content}
      </div>
      {/* <ActionBar text={message.content} /> */}
    </div>
  );
};
