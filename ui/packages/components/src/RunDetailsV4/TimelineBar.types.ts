/**
 * Type definitions for the composable TimelineBar component.
 * Feature: 001-composable-timeline-bar
 */

import type { ReactNode } from 'react';

// ============================================================================
// Core Component Types
// ============================================================================

/**
 * Available fill patterns for bars.
 */
export type BarPattern = 'solid' | 'barber-pole' | 'dotted';

/**
 * Available icons for bars.
 */
export type BarIcon =
  | 'gear' // INNGEST timing
  | 'building' // SERVER timing
  | 'lightning' // CONNECTING timing
  | 'function' // step.run
  | 'clock' // step.sleep
  | 'mail' // step.waitForEvent
  | 'arrow' // step.invoke
  | 'checkbox' // root run (completed)
  | 'close-circle' // root run (failed)
  | 'stop-circle' // root run (cancelled)
  | 'none';

/**
 * Style keys for different bar types.
 * Each key maps to a BarStyle configuration.
 * Colors use status-based semantics for consistency (e.g., green for completed).
 */
export type BarStyleKey =
  // Root run bar (status color, checkbox icon)
  | 'root'
  // Step types - step.run uses status color, others use fallback (pending design)
  | 'step.run'
  | 'step.sleep'
  | 'step.waitForEvent'
  | 'step.invoke'
  // Timing categories
  | 'timing.inngest' // Queue time (short, gray)
  | 'timing.server' // Execution time (tall, barber-pole, status color)
  | 'timing.connecting' // Connection time (short, dotted border, status color)
  // HTTP timing phases (nested under server execution)
  | 'timing.http.dns' // DNS lookup (short, sky)
  | 'timing.http.tcp' // TCP connection (short, cyan)
  | 'timing.http.tls' // TLS handshake (short, teal)
  | 'timing.http.server' // Server processing / TTFB (short, emerald)
  | 'timing.http.transfer' // Content transfer (short, green)
  // Generic fallback
  | 'default';

/**
 * Bar height variants.
 */
export type BarHeight = 'short' | 'tall';

/**
 * Visual style configuration for a bar type.
 */
export interface BarStyle {
  /** Tailwind background color class for the bar */
  barColor: string;

  /** Tailwind text color class (optional, defaults to standard) */
  textColor?: string;

  /** Tailwind text color class for the duration label (optional, defaults to textColor) */
  durationColor?: string;

  /** Icon to display for this style */
  icon?: BarIcon;

  /** Fill pattern for the bar */
  pattern?: BarPattern;

  /** Label format (for timing bars) */
  labelFormat?: 'uppercase' | 'titlecase' | 'default';

  /** Bar height variant (defaults to 'tall') */
  barHeight?: BarHeight;

  /** Whether this bar should use status-based coloring (green/red/etc based on run status) */
  statusBased?: boolean;
}

/**
 * A segment within a compound bar (used for inline timing visualization).
 */
export interface BarSegment {
  /** Unique identifier for the segment */
  id: string;

  /** Start position as percentage within the bar (0-100) */
  startPercent: number;

  /** Width as percentage within the bar (0-100) */
  widthPercent: number;

  /** Style key for this segment */
  style: BarStyleKey;

  /** Optional tooltip content */
  tooltip?: string;

  /** Run status for status-based coloring (e.g., COMPLETED, FAILED, CANCELLED) */
  status?: string;
}

/**
 * Props for the TimelineBar component.
 * Renders a single row in the timeline with optional expansion and nested children.
 */
export interface TimelineBarProps {
  /** Display name shown in the left panel */
  name: string;

  /** Duration in milliseconds */
  duration: number;

  /** Optional icon displayed before the name */
  icon?: BarIcon;

  /** Start position as percentage of timeline width (0-100) */
  startPercent: number;

  /** Width as percentage of timeline width (0-100) */
  widthPercent: number;

  /** Nesting depth (0 = root level) */
  depth: number;

  /** Column divider position - width of left panel as percentage (0-100) */
  leftWidth: number;

  /** Style key for visual appearance */
  style: BarStyleKey;

  /** Optional segments within the bar (for compound bars) */
  segments?: BarSegment[];

  /** Whether this bar can be expanded to show children */
  expandable?: boolean;

  /** Current expansion state (controlled) */
  expanded?: boolean;

  /** Callback when expand toggle is clicked */
  onToggle?: () => void;

  /** Callback when the row is clicked (for selection) */
  onClick?: () => void;

  /** Whether this bar is currently selected */
  selected?: boolean;

  /** Child bars to render when expanded */
  children?: ReactNode;

  /** Optional organization name for SERVER timing label */
  orgName?: string;

  /** Run status for status-based coloring (e.g., COMPLETED, FAILED, CANCELLED) */
  status?: string;

  /**
   * View offset - start position as percentage (0-100).
   * Used for zooming: defines where the visible window starts in the full timeline.
   * @default 0
   */
  viewStartOffset?: number;

  /**
   * View offset - end position as percentage (0-100).
   * Used for zooming: defines where the visible window ends in the full timeline.
   * @default 100
   */
  viewEndOffset?: number;

  /** Start time of this bar (for hover tooltip) */
  startTime?: Date;

  /** End time of this bar, null if in progress (for hover tooltip) */
  endTime?: Date | null;

  /** Timeline start time, used to compute offsets in hover tooltip */
  minTime?: Date;
}

// ============================================================================
// Data Types for Timeline Container
// ============================================================================

/**
 * Complete timeline data for rendering.
 */
export interface TimelineData {
  /** Overall timeline bounds */
  minTime: Date;
  maxTime: Date;

  /** Root-level bars (steps) */
  bars: TimelineBarData[];

  /** Column divider position */
  leftWidth: number;

  /** Optional organization name */
  orgName?: string;
}

/**
 * Data for a single bar in the timeline.
 */
export interface TimelineBarData {
  /** Unique identifier */
  id: string;

  /** Display name */
  name: string;

  /** Start time */
  startTime: Date;

  /** End time (null if in progress) */
  endTime: Date | null;

  /** Bar style */
  style: BarStyleKey;

  /** Nested child bars */
  children?: TimelineBarData[];

  /** Timing breakdown data (for expandable bars) */
  timingBreakdown?: TimingBreakdownData;

  /** HTTP timing breakdown (for steps with HTTP timing metadata) */
  httpTimingBreakdown?: HTTPTimingBreakdownData;

  /** Whether this bar represents the root run (clicking shows TopInfo) */
  isRoot?: boolean;

  /** Run status for status-based coloring (e.g., COMPLETED, FAILED, CANCELLED) */
  status?: string;
}

/**
 * Timing breakdown for a step bar.
 */
export interface TimingBreakdownData {
  /** Queue time (INNGEST) */
  queueMs: number;

  /** Execution time (SERVER) */
  executionMs: number;

  /** Total duration */
  totalMs: number;
}

/**
 * HTTP timing breakdown for a step bar.
 * Represents the sequential phases of an HTTP request lifecycle
 * (Inngest's HTTP call to the SDK endpoint).
 */
export interface HTTPTimingBreakdownData {
  /** Time spent resolving the domain name */
  dnsLookupMs: number;

  /** Time spent establishing the TCP connection */
  tcpConnectionMs: number;

  /** Time spent on TLS negotiation */
  tlsHandshakeMs: number;

  /** Time from request sent to first byte received (TTFB) */
  serverProcessingMs: number;

  /** Time spent downloading the response body */
  contentTransferMs: number;

  /** Total request duration */
  totalMs: number;
}
