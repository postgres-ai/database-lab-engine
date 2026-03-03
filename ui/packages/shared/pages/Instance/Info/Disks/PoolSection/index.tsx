import { makeStyles } from '@material-ui/core'

import { colors } from '@postgres.ai/shared/styles/colors'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { Status as PerformanceStatus } from '@postgres.ai/shared/components/Status'

import { ProgressBar } from '../Disk/ProgressBar'
import { DatasetRow, DatasetInfo } from './DatasetRow'

const WARNING_THRESHOLD_PERCENT = 80

type Props = {
  poolName: string
  totalSize: number
  freeSize: number
  datasets: DatasetInfo[]
}

const useStyles = makeStyles(
  {
    root: {
      border: `1px solid ${colors.consoleStroke}`,
      padding: '6px 8px 8px',
      borderRadius: '4px',
      backgroundColor: colors.consoleMenuBackground,

      '& + $root': {
        marginTop: '8px',
      },
    },
    headerRow: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'baseline',
      marginBottom: '4px',
    },
    poolName: {
      fontWeight: 700,
      fontSize: '14px',
    },
    poolSize: {
      fontSize: '13px',
      fontWeight: 700,
    },
    progressBarWrapper: {
      marginTop: '4px',
    },
    poolStats: {
      fontSize: '12px',
      marginTop: '2px',
    },
    warningMessage: {
      fontSize: '10px',
      marginTop: '4px',
    },
    datasets: {
      marginTop: '8px',
      display: 'flex',
      flexDirection: 'column',
      gap: '6px',
    },
  },
  { index: 1 },
)

const getPercent = (value: number, total: number) =>
  total === 0 ? 0 : Math.round((value / total) * 100)

export const PoolSection = (props: Props) => {
  const classes = useStyles()
  const { poolName, totalSize, freeSize, datasets } = props

  const usedSize = Math.max(0, totalSize - freeSize)
  const usedPercent = getPercent(usedSize, totalSize)
  const freePercent = Math.min(100, getPercent(freeSize, totalSize))
  const hasActiveDataset = datasets.some((d) => d.status === 'active')
  const shouldShowWarning =
    hasActiveDataset && usedPercent > WARNING_THRESHOLD_PERCENT

  return (
    <div className={classes.root}>
      <div className={classes.headerRow}>
        <span className={classes.poolName}>{poolName}</span>
        <span className={classes.poolSize}>{formatBytesIEC(totalSize)}</span>
      </div>
      <div className={classes.progressBarWrapper}>
        <ProgressBar
          value={usedSize}
          total={totalSize}
          thresholdPercent={WARNING_THRESHOLD_PERCENT}
          style={{ marginTop: 0 }}
        />
      </div>
      <div className={classes.poolStats}>
        <strong>{formatBytesIEC(usedSize)}</strong> used ({usedPercent}%) · {freePercent}% free
      </div>

      {shouldShowWarning && (
        <PerformanceStatus type="warning" className={classes.warningMessage}>
          +{WARNING_THRESHOLD_PERCENT}% disk usage may result in performance
          degradation
        </PerformanceStatus>
      )}

      <div className={classes.datasets}>
        {datasets.map((dataset) => (
          <DatasetRow key={dataset.id ?? dataset.name} {...dataset} />
        ))}
      </div>
    </div>
  )
}
