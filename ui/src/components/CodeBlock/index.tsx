import { useState } from 'react';

import classNames from '../../utils/classnames';
import CopyButton from './CopyButton';
import { SyntaxHighlight } from './SyntaxHighlight';

interface CodeBlockProps {
  tabs: {
    label: string;
    content: string;
  }[];
  expanded?: boolean;
  modal?: (...args: any[]) => any;
}

export default function CodeBlock({ tabs, modal, expanded = false }: CodeBlockProps) {
  const [activeTab, setActiveTab] = useState(0);

  const handleTabClick = (index) => {
    setActiveTab(index);
  };

  const handleCopyClick = () => {
    navigator.clipboard.writeText(tabs[activeTab].content);
  };

  const handleExpandClick = () => {
    modal?.(tabs);
  };

  return (
    <div className="w-full bg-slate-800/30 border border-slate-700/30 rounded-lg shadow overflow-hidden">
      <div className="bg-slate-800/40 flex justify-between shadow border-b border-slate-700/20">
        <div className="flex -mb-px">
          {tabs.map((tab, i) => (
            <button
              className={classNames(
                i === activeTab
                  ? `border-indigo-400 text-white`
                  : `border-transparent text-slate-400`,
                `text-xs px-5 py-2.5 border-b block transition-all duration-150 outline-none`,
              )}
              onClick={() => handleTabClick(i)}
              key={i}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="flex gap-2 items-center mr-2">
          <CopyButton btnAction={handleCopyClick} />
          {/* <Button
            label={expanded ? "Close" : "Expand"}
            btnAction={handleExpandClick}
          /> */}
        </div>
      </div>
      <div
        className={classNames(
          expanded ? `max-w-[800px] max-h-[800px]` : `max-h-[300px]`,
          `overflow-scroll grid`,
        )}
      >
        {tabs.map((tab, i) => (
          <code
            className={classNames(
              i === activeTab ? ` ` : `opacity-0 pointer-events-none`,
              `col-start-1 row-start-1 transition-all duration-150`,
            )}
          >
            <SyntaxHighlight code={tabs[i].content} className="p-4 text-2xs" />
          </code>
        ))}
      </div>
    </div>
  );
}
