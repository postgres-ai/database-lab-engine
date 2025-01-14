export const compareSnapshotsDesc = (
  a: { dataStateAtDate: Date },
  b: { dataStateAtDate: Date },
): number => {
  const { dataStateAtDate: dateA } = a
  const { dataStateAtDate: dateB } = b
  return dateB.getTime() - dateA.getTime()
}
