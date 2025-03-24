import parse, { IPostgresInterval } from "postgres-interval"

export function formatPostgresInterval(balance: string): string {
    const interval: IPostgresInterval = parse(balance);

    const units: Partial<Record<keyof Omit<IPostgresInterval, 'toPostgres' | 'toISO' | 'toISOString' | 'toISOStringShort'>, string>> = {
        years: 'y',
        months: 'mo',
        days: 'd',
        hours: 'h',
        minutes: 'm',
        seconds: 's',
        milliseconds: 'ms',
    };

    const sign = Object.keys(units)
        .map((key) => interval[key as keyof IPostgresInterval] || 0)
        .find((value) => value !== 0) ?? 0;

    const isNegative = sign < 0;

    const formattedParts = (Object.keys(units) as (keyof typeof units)[])
        .map((key) => {
            const value = interval[key];
            return value && Math.abs(value) > 0 ? `${Math.abs(value)}${units[key]}` : null;
        })
        .filter(Boolean);

    return (isNegative ? '-' : '') + formattedParts.join(' ');
}