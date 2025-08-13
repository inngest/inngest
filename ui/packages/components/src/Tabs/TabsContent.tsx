import * as TabsPrimitive from '@radix-ui/react-tabs';

interface TabsContentProps extends React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content> {}

export function TabsContent({ children, className, ...props }: TabsContentProps) {
  return (
    <TabsPrimitive.Content className={className} {...props}>
      {children}
    </TabsPrimitive.Content>
  );
}
