'use client';

import { useState } from 'react';
import { RiFileCopyLine } from '@remixicon/react';

// const CopyButton = ({ text }: { text: string }) => {
//   const [isCopied, setIsCopied] = useState(false);

//   const handleCopy = () => {
//     navigator.clipboard
//       .writeText(text)
//       .then(() => {
//         setIsCopied(true);
//         setTimeout(() => setIsCopied(false), 2000);
//       })
//       .catch((err) => {
//         console.error('Failed to copy text: ', err);
//       });
//   };

//   return (
//     <button
//       onClick={handleCopy}
//       className="rounded-md p-1 text-gray-500 hover:text-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400"
//       aria-label="Copy message"
//     >
//       {isCopied ? 'Copied!' : <RiFileCopyLine className="size-4" />}
//     </button>
//   );
// };

// const ActionBar = ({ text }: { text: string }) => {
//   return (
//     <div className="absolute -bottom-8 right-0 flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
//       <CopyButton text={text} />
//     </div>
//   );
// };

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
