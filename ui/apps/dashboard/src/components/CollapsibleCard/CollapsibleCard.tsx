import { forwardRef } from 'react';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { AnimatePresence, motion } from 'framer-motion';

export const CollapsibleCardRoot = AccordionPrimitive.Root;

const CollapsibleCardItem = forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Item>
>(({ children, ...props }, forwardedRef) => {
  return (
    <AccordionPrimitive.Item
      {...props}
      ref={forwardedRef}
      className="border-subtle bg-canvasBase rounded-md border"
    >
      {children}
    </AccordionPrimitive.Item>
  );
});

export const CollapsibleCardHeader = AccordionPrimitive.Header;
export const CollapsibleCardTrigger = AccordionPrimitive.Trigger;
export const CollapsibleCardContentWrapper = AnimatePresence;

const CollapsibleCardContent = forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Content>
>(({ children, ...props }, forwardedRef) => {
  return (
    <AccordionPrimitive.Content {...props} ref={forwardedRef} forceMount>
      <motion.div
        initial={{ y: 0, opacity: 0.2 }}
        animate={{ y: 0, opacity: 1 }}
        exit={{
          y: 0,
          opacity: 0.2,
          transition: { duration: 0.2, type: 'tween' },
        }}
        transition={{
          duration: 0.15,
          type: 'tween',
        }}
      >
        {children}
      </motion.div>
    </AccordionPrimitive.Content>
  );
});

CollapsibleCardItem.displayName = AccordionPrimitive.Item.displayName;
CollapsibleCardContent.displayName = AccordionPrimitive.Content.displayName;

export { CollapsibleCardItem, CollapsibleCardContent };
