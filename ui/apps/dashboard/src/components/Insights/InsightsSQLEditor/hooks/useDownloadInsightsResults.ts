import { useCallback } from 'react';

import type { InsightsFetchResult } from '../../InsightsStateMachineContext/types';

/**
 * Triggers a file download in the browser by creating a temporary blob URL
 * and programmatically clicking a download link.
 *
 * @param content - The file content as a string
 * @param filename - The name for the downloaded file
 * @param contentType - The MIME type of the file (e.g., 'text/csv', 'application/json')
 */
function downloadFile(content: string, filename: string, contentType: string) {
  const blob = new Blob([content], { type: contentType });
  const url = URL.createObjectURL(blob);

  try {
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  } finally {
    // Always clean up the blob URL to prevent memory leaks
    URL.revokeObjectURL(url);
  }
}

/**
 * Determines if a CSV value needs to be quoted according to RFC 4180.
 * Values containing commas, quotes, newlines, or leading/trailing spaces must be quoted.
 *
 * @param value - The string value to check
 * @returns True if the value needs quoting, false otherwise
 */
function needsCSVQuoting(value: string): boolean {
  return (
    value.includes(',') ||
    value.includes('"') ||
    value.includes("'") ||
    value.includes('\n') ||
    value.includes('\r') ||
    value.startsWith(' ') ||
    value.endsWith(' ')
  );
}

/**
 * Escapes and quotes a CSV value according to RFC 4180.
 * Double quotes within the value are escaped by doubling them.
 *
 * @param value - The string value to escape and quote
 * @returns The escaped value wrapped in double quotes
 */
function escapeCSVValue(value: string): string {
  // Escape double quotes by doubling them per RFC 4180
  return `"${value.replace(/"/g, '""')}"`;
}

/**
 * Converts Insights query results to CSV format.
 * Handles null values, dates (converted to ISO 8601), and special characters.
 *
 * @param data - The Insights query results containing columns and rows
 * @returns A CSV-formatted string with headers and data rows
 */
function convertToCSV(data: InsightsFetchResult): string {
  const { columns, rows } = data;

  // Create header row
  const headers = columns.map((col) => col.name);
  const csvRows = [headers.join(',')];

  // Add data rows
  rows.forEach((row) => {
    const values = headers.map((header) => {
      const value = row.values[header];

      // Handle null/undefined
      if (value === null || value === undefined) {
        return '';
      }

      // Handle dates
      if (value instanceof Date) {
        return value.toISOString();
      }

      // Handle strings that need quoting per RFC 4180
      const stringValue = String(value);

      if (needsCSVQuoting(stringValue)) {
        return escapeCSVValue(stringValue);
      }

      return stringValue;
    });

    csvRows.push(values.join(','));
  });

  return csvRows.join('\n');
}

/**
 * Converts Insights query results to JSON format.
 * Extracts row values and converts Date objects to ISO 8601 strings.
 *
 * @param data - The Insights query results containing columns and rows
 * @returns A pretty-printed JSON string (2-space indentation) containing an array of objects
 */
function convertToJSON(data: InsightsFetchResult): string {
  const { rows } = data;

  // Extract just the values from each row, converting Dates to ISO strings
  const jsonData = rows.map((row) => row.values);

  // JSON.stringify handles Date objects automatically, but we use a replacer
  // to ensure consistent ISO string formatting
  return JSON.stringify(
    jsonData,
    (_key, value) => (value instanceof Date ? value.toISOString() : value),
    2,
  );
}

/**
 * Generates a timestamp string suitable for use in filenames.
 * Format: YYYY-MM-DDTHH-mm-ss (colons and dots replaced with hyphens)
 *
 * @returns A filesystem-safe timestamp string
 */
function generateTimestamp(): string {
  return new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5);
}

/**
 * Sanitizes a query name for use in filenames by replacing whitespace with underscores
 * and removing any characters that are not alphanumeric or underscores.
 *
 * @param queryName - The query name to sanitize
 * @returns A filesystem-safe query name containing only A-Z, a-z, 0-9, and underscores
 * @example
 * sanitizeQueryName('My Query!') // Returns: "My_Query"
 * sanitizeQueryName('Test @#$ Query') // Returns: "Test_Query"
 */
function sanitizeQueryName(queryName: string): string {
  return (
    queryName
      .trim()
      .replace(/\s+/g, '_') // Replace whitespace with underscores
      .replace(/[^A-Za-z0-9_]/g, '') || 'insights-query' // Remove non-alphanumeric chars except underscores
  );
}

export type DownloadFormat = 'csv' | 'json';

/**
 * Custom hook that provides download functionality for Insights query results.
 * Supports both CSV and JSON export formats with proper data formatting.
 *
 * @param data - The Insights query results to download, or undefined if no data available
 * @param queryName - The name of the query to use in the filename
 * @returns An object containing download functions:
 *   - downloadAsCSV: Downloads data as a CSV file
 *   - downloadAsJSON: Downloads data as a JSON file
 *   - download: Generic download function accepting format parameter
 */
export function useDownloadInsightsResults(
  data: InsightsFetchResult | undefined,
  queryName?: string,
) {
  const downloadAsCSV = useCallback(() => {
    if (!data) return;

    const csv = convertToCSV(data);
    const filename = sanitizeQueryName(queryName || 'insights-query');
    const timestamp = generateTimestamp();
    downloadFile(csv, `${filename}-${timestamp}.csv`, 'text/csv');
  }, [data, queryName]);

  const downloadAsJSON = useCallback(() => {
    if (!data) return;

    const json = convertToJSON(data);
    const filename = sanitizeQueryName(queryName || 'insights-query');
    const timestamp = generateTimestamp();
    downloadFile(json, `${filename}-${timestamp}.json`, 'application/json');
  }, [data, queryName]);

  const download = useCallback(
    (format: DownloadFormat) => {
      if (format === 'csv') {
        downloadAsCSV();
      } else {
        downloadAsJSON();
      }
    },
    [downloadAsCSV, downloadAsJSON],
  );

  return {
    downloadAsCSV,
    downloadAsJSON,
    download,
  };
}
