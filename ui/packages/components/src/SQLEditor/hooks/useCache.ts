import { useRef } from 'react';

export interface CacheEntry<T> {
  data: T;
  timestamp: number;
}

export interface CacheOptions {
  ttl: number; // Time to live in milliseconds
  name?: string; // Optional name for debugging/identification
}

export interface Cache<T> {
  get: (key: string) => T | null;
  set: (key: string, data: T) => void;
  has: (key: string) => boolean;
  clear: () => void;
}

/**
 * A generic cache hook that stores data with a TTL (time to live).
 * The cache is synchronous and can be used with any type of data.
 *
 * @param options - Cache configuration options
 * @returns Cache object with get, set, has, and clear methods
 *
 * @example
 * const cache = useCache<string[]>({ ttl: 5 * 60 * 1000 }); // 5 minutes
 *
 * // Set data
 * cache.set('myKey', ['value1', 'value2']);
 *
 * // Get data (returns null if expired or not found)
 * const data = cache.get('myKey');
 *
 * // Check if key exists and is valid
 * if (cache.has('myKey')) {
 *   // Do something
 * }
 *
 * // Clear all cache entries
 * cache.clear();
 */
export function useCache<T>(options: CacheOptions): Cache<T> {
  const { ttl } = options;
  const cacheRef = useRef<Map<string, CacheEntry<T>>>(new Map());

  return {
    get: (key: string): T | null => {
      const entry = cacheRef.current.get(key);
      if (!entry) {
        return null;
      }

      const isExpired = Date.now() - entry.timestamp > ttl;
      if (isExpired) {
        cacheRef.current.delete(key);
        return null;
      }

      return entry.data;
    },

    set: (key: string, data: T): void => {
      cacheRef.current.set(key, {
        data,
        timestamp: Date.now(),
      });
    },

    has: (key: string): boolean => {
      const entry = cacheRef.current.get(key);
      if (!entry) {
        return false;
      }

      const isExpired = Date.now() - entry.timestamp > ttl;
      if (isExpired) {
        cacheRef.current.delete(key);
        return false;
      }

      return true;
    },

    clear: (): void => {
      cacheRef.current.clear();
    },
  };
}
