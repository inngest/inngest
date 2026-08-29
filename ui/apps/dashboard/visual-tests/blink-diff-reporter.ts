import { execFileSync } from 'node:child_process';
import { existsSync, readdirSync } from 'node:fs';
import path from 'node:path';

import type { FullConfig, Reporter } from '@playwright/test/reporter';

const actualSuffix = '-actual.png';

function findActualScreenshots(directory: string): string[] {
  if (!existsSync(directory)) {
    return [];
  }

  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const entryPath = path.join(directory, entry.name);

    if (entry.isDirectory()) {
      return findActualScreenshots(entryPath);
    }

    return entry.name.endsWith(actualSuffix) ? [entryPath] : [];
  });
}

function imageMagickCommand(): string {
  for (const command of ['magick', 'convert']) {
    try {
      execFileSync(command, ['-version'], { stdio: 'ignore' });
      return command;
    } catch {
      // Try the next ImageMagick executable name.
    }
  }

  throw new Error('ImageMagick is required to create animated visual diffs');
}

function createBlinkDiff(command: string, actualPath: string): void {
  const expectedPath = actualPath.replace(actualSuffix, '-expected.png');
  if (!existsSync(expectedPath)) {
    return;
  }

  const outputPath = actualPath.replace(actualSuffix, '-blink.gif');
  const labelArguments = (imagePath: string, label: string) => [
    '(',
    imagePath,
    '-gravity',
    'north',
    '-background',
    '#ffffff',
    '-splice',
    '0x30',
    '-fill',
    '#111827',
    '-font',
    'DejaVu-Sans',
    '-pointsize',
    '18',
    '-annotate',
    '+0+6',
    label,
    ')',
  ];

  execFileSync(
    command,
    [
      '-delay',
      '80',
      ...labelArguments(expectedPath, 'Expected'),
      ...labelArguments(actualPath, 'Actual'),
      '-loop',
      '0',
      outputPath,
    ],
    { stdio: 'ignore' },
  );

  console.log(
    `Animated visual diff: ${path.relative(process.cwd(), outputPath)}`,
  );
}

export default class BlinkDiffReporter implements Reporter {
  private outputDirectories: string[] = [];

  onBegin(config: FullConfig): void {
    this.outputDirectories = [
      ...new Set(config.projects.map((project) => project.outputDir)),
    ];
  }

  onEnd(): void {
    const actualScreenshots = this.outputDirectories.flatMap(
      findActualScreenshots,
    );
    if (actualScreenshots.length === 0) {
      return;
    }

    try {
      const command = imageMagickCommand();
      actualScreenshots.forEach((actualPath) =>
        createBlinkDiff(command, actualPath),
      );
    } catch (error) {
      console.warn(`Unable to create animated visual diffs: ${String(error)}`);
    }
  }
}
