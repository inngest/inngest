import { useCallback, useLayoutEffect, useRef, useState } from 'react';

export const useStickToBottom = () => {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [isAtBottom, setIsAtBottom] = useState(true);

  const scrollToBottom = useCallback(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, []);

  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (!el) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = el;
      // A little bit of tolerance
      const isAtBottom = scrollHeight - scrollTop - clientHeight < 20;
      setIsAtBottom(isAtBottom);
    };

    const observer = new MutationObserver(() => {
      if (isAtBottom) {
        scrollToBottom();
      }
    });

    el.addEventListener('scroll', handleScroll, { passive: true });
    observer.observe(el, { childList: true, subtree: true });

    return () => {
      el.removeEventListener('scroll', handleScroll);
      observer.disconnect();
    };
  }, [isAtBottom, scrollToBottom]);

  return { scrollRef, isAtBottom, scrollToBottom };
};
