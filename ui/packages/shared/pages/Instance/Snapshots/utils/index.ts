import { Snapshot } from 'types/api/entities/snapshot'
import { isSameDayUTC } from '@postgres.ai/shared/utils/date'

export const generateSnapshotPageId = (id: string) => {
  if (!id.includes('@')) return null

  const snapshotIdPart = id.split('@')[1]
  return snapshotIdPart.startsWith('snapshot_')
    ? snapshotIdPart.substring(9)
    : snapshotIdPart
}

export const groupSnapshotsByCreatedAtDate = (snapshots: Snapshot[]) => {
  const groups: Snapshot[][] = []

  snapshots.forEach((snapshot) => {
    let grouped = false

    for (const group of groups) {
      const groupDate = new Date(group[0].createdAtDate)

      if (isSameDayUTC(snapshot.createdAtDate, groupDate)) {
        group.push(snapshot)
        grouped = true
        break
      }
    }

    if (!grouped) {
      groups.push([snapshot])
    }
  })

  groups.sort((a, b) => {
    return b[0].createdAtDate.getTime() - a[0].createdAtDate.getTime()
  })

  return groups
}
