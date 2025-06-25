export const compareSnapshotsDesc = (
  a: { dataStateAtDate: Date },
  b: { dataStateAtDate: Date },
): number => {
  const dataStateAtDateA = a.dataStateAtDate?.getTime() ?? 0
  const dataStateAtDateB = b.dataStateAtDate?.getTime() ?? 0
  return dataStateAtDateB - dataStateAtDateA
}
