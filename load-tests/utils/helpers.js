/**
 * General utility functions for k6 load testing scripts.
 */

import { check } from 'k6';

/**
 * Builds HTTP request headers with JSON content-type, optional Bearer token, and correlation ID.
 */
export function buildHeaders(token = null, customHeaders = {}) {
  const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
    'X-Request-ID': `k6-${Date.now()}-${Math.floor(Math.random() * 1000000)}`,
    ...customHeaders,
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  return headers;
}

/**
 * Validates standard JSON response contract for Vyavsa.
 */
export function checkResponse(res, expectedStatus = 200, checkName = 'response valid') {
  return check(res, {
    [`${checkName} status ${expectedStatus}`]: (r) => r.status === expectedStatus,
    [`${checkName} non-empty body`]: (r) => r.body && r.body.length > 0,
  });
}

/**
 * Generates random integer between min and max inclusive.
 */
export function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

/**
 * Selects a random item from an array.
 */
export function randomChoice(arr) {
  if (!arr || arr.length === 0) return null;
  return arr[Math.floor(Math.random() * arr.length)];
}

/**
 * Generates an alphanumeric pseudo-random string of given length.
 */
export function randomString(length = 8) {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

/**
 * Generates a mock Indian 10-digit phone number.
 */
export function randomPhone() {
  return `+9198${randomInt(10000000, 99999999)}`;
}

/**
 * Safe JSON parser.
 */
export function parseJSON(body) {
  try {
    return JSON.parse(body);
  } catch (e) {
    return null;
  }
}
