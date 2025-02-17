import { RiMoonClearFill, RiSunLine, RiWindow2Line } from '@remixicon/react';
import { useTheme } from 'next-themes';

import SegmentedControl from '../SegmentedControl/SegmentedControl';

export default function ModeSwitch() {
  const { theme, setTheme } = useTheme();

  return (
    <SegmentedControl defaultValue={theme}>
      <SegmentedControl.Button
        value="light"
        icon={<RiSunLine />}
        onClick={() => setTheme('light')}
      />
      <SegmentedControl.Button
        value="dark"
        icon={<RiMoonClearFill />}
        onClick={() => setTheme('dark')}
      />
      <SegmentedControl.Button
        value="system"
        icon={<RiWindow2Line className="rotate-180" />}
        onClick={() => setTheme('system')}
      />
    </SegmentedControl>
  );
}
