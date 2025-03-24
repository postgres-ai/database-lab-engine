/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import parse, { IPostgresInterval } from "postgres-interval";

export const generateToken = function () {
  const a =
    'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890'.split('')
  const b = []

  for (let i = 0; i < 32; i++) {
    const j = (Math.random() * (a.length - 1)).toFixed(0)
    b[i] = a[j as keyof typeof a]
  }

  return b.join('')
}

export const isHttps = function (url: string | string[]) {
  return url && url.length > 5 && url.indexOf('https') === 0
}

export const snakeToCamel = (str: string) =>
  str.replace(/([-_]\w)/g, (g) => g[1].toUpperCase())

export const validateDLEName = (name: string) => {
  return (
    name.length > 0 &&
    !name.match(/^([a-z](?:[-a-z0-9]{0,61}[a-z0-9])?|[1-9][0-9]{0,19})$/)
  )
}

export const isMobileDevice = (): boolean => {
  let hasTouchScreen = false;

  // Check for modern touch screen devices using maxTouchPoints
  if ("maxTouchPoints" in navigator) {
    hasTouchScreen = navigator.maxTouchPoints > 0;
  }
  // Check for older versions of IE with msMaxTouchPoints
  else if ("msMaxTouchPoints" in navigator) {
    hasTouchScreen = (navigator as unknown as { msMaxTouchPoints: number }).msMaxTouchPoints > 0;
  }
  // Use matchMedia to check for coarse pointer devices
  else {
    const mQ = window.matchMedia("(pointer:coarse)");
    if (mQ && mQ.media === "(pointer:coarse)") {
      hasTouchScreen = mQ.matches;
    }
    // Check for the presence of the orientation property as a fallback (deprecated in modern browsers)
    else if ('orientation' in window) {
      hasTouchScreen = true;
    }
    // Last resort: fallback with user agent sniffing
    else {
      const UA = navigator.userAgent;
      hasTouchScreen = (
        /\b(BlackBerry|webOS|iPhone|IEMobile)\b/i.test(UA) ||
        /\b(Android|Windows Phone|iPad|iPod)\b/i.test(UA)
      );
    }
  }

  // Check for mobile screen width, 1366 because of iPad Pro in Landscape mode
  // If this is not necessary, may reduce value to 1024 or 768
  const isMobileScreen = window.innerWidth <= 1366;

  return hasTouchScreen && isMobileScreen;
}

export const pgIntervalToHours = (interval?: string | null): number | null => {
  if (!interval) {
    return null;
  }

  const parsed: IPostgresInterval = parse(interval);

  const yearsToHours = (parsed.years ?? 0) * 365 * 24;
  const monthsToHours = (parsed.months ?? 0) * 30 * 24;
  const daysToHours = (parsed.days ?? 0) * 24;
  const hours = parsed.hours ?? 0;
  const minutesToHours = (parsed.minutes ?? 0) / 60;
  const secondsToHours = (parsed.seconds ?? 0) / 3600;

  return yearsToHours + monthsToHours + daysToHours + hours + minutesToHours + secondsToHours;
}

export const hoursToPgInterval = (hours: number): string => {
  const totalMinutes = Math.floor(hours * 60);
  const days = Math.floor(totalMinutes / (24 * 60));
  const remainingMinutes = totalMinutes % (24 * 60);
  const h = Math.floor(remainingMinutes / 60);
  const m = remainingMinutes % 60;

  if (days > 0) {
    return `${days} days ${h}:${m}:00`;
  } else {
    return `${h}:${m}:00`;
  }
}