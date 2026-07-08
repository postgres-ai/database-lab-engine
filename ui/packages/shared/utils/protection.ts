// protectionOptions lists the deletion-protection choices shared by the branch and snapshot
// detail pages: no protection, a set of bounded durations, and forever. The values are the
// protectionDurationMinutes passed to the update endpoints ('0' means forever, 'none' clears
// protection).
export const protectionOptions = [
  { value: 'none', children: 'No protection' },
  { value: '60', children: '1 hour' },
  { value: '720', children: '12 hours' },
  { value: '1440', children: '1 day' },
  { value: '4320', children: '3 days' },
  { value: '10080', children: '7 days' },
  { value: '43200', children: '30 days' },
  { value: '0', children: 'Forever' },
]

// getProtectionSelectValue maps an entity's protection state to the Select value: 'none' when
// unprotected, 'current' when protected until a specific time, '0' when protected forever.
export const getProtectionSelectValue = (
  isProtected: boolean,
  protectedTillDate: Date | null,
): string => {
  if (!isProtected) return 'none'
  if (!protectedTillDate) return '0'
  return 'current'
}
