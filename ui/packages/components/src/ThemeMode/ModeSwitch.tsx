import { RiMoonClearFill, RiSunLine, RiWindow2Line } from '@remixicon/react';

import SegmentedControl from '../SegmentedControl/SegmentedControl';

export default function ModeSwitch() {
  const handleThemeChange = (newTheme: string) => {
    console.log(newTheme);
    // TODO: Implement API call to update user's theme preference
  };

  return (
    <SegmentedControl defaultValue="light">
      <SegmentedControl.Button
        value="light"
        icon={<RiSunLine />}
        onClick={() => handleThemeChange('light')}
      />
      <SegmentedControl.Button
        value="dark"
        icon={<RiMoonClearFill />}
        onClick={() => handleThemeChange('dark')}
      />
      <SegmentedControl.Button
        value="system"
        icon={<RiWindow2Line className="rotate-180" />}
        onClick={() => handleThemeChange('system')}
      />
    </SegmentedControl>
  );
}
