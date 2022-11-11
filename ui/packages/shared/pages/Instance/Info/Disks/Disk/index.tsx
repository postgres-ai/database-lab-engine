import { makeStyles } from '@material-ui/core'
import { formatDistanceToNowStrict } from 'date-fns'

import { colors } from '@postgres.ai/shared/styles/colors'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { Status as PerformanceStatus } from '@postgres.ai/shared/components/Status'
import { formatUTC } from '@postgres.ai/shared/utils/date'

import { Property } from '../../components/Property'
import { ActionsMenu } from './ActionsMenu'
import { Status } from './Status'
import { Marker } from './Marker'
import { ProgressBar } from './ProgressBar'

const WARNING_THRESHOLD_PERCENT = 80

type Props = {
  id: string | null
  name: string
  totalDataSize: number
  mode: string
  refreshingStartDate: Date | null
  clonesCount: number
  snapshotsCount: number
  usedDataSize: number
  freeDataSize: number
  status: 'refreshing' | 'active' | 'empty'
}

const useStyles = makeStyles(
  {
    root: {
      border: `1px solid ${colors.consoleStroke}`,
      padding: '6px 8px 8px',
      borderRadius: '4px',

      '& + $root': {
        marginTop: '8px',
      },
    },
    header: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
    },
    titleWrapper: {
      display: 'flex',
      flex: '1 1 auto',
      alignItems: 'center',
      marginRight: '16px',
      minWidth: 0,
    },
    title: {
      fontWeight: 700,
      fontSize: '14px',
      margin: '0 4px 0 0',
      whiteSpace: 'nowrap',
      textOverflow: 'ellipsis',
      overflow: 'hidden',
    },
    content: {
      marginTop: '8px',
    },
    markerUsed: {
      color: colors.primary.light,
    },
    markerFree: {
      color: colors.gray,
    },
    warningMessage: {
      fontSize: '10px',
      marginTop: '6px',
    },
    uppercaseContent: {
      textTransform: 'uppercase',
    },
  },
  { index: 1 },
)

const getPercent = (value: number, total: number) =>
  Math.round((value / total) * 100)

export const Disk = (props: Props) => {
  const classes = useStyles()

  const shouldShowWarning =
    props.status === 'active' &&
    getPercent(props.usedDataSize, props.totalDataSize) >
      WARNING_THRESHOLD_PERCENT

  return (
    <div className={classes.root}>
      <div className={classes.header}>
        <div className={classes.titleWrapper}>
          <h6 title={props.name} className={classes.title}>
            {props.name}
          </h6>
          <ActionsMenu
            poolId={props.id}
            poolName={props.name}
            isActive={props.status === 'active'}
          />
        </div>

        <Status value={props.status} hasWarning={shouldShowWarning} />
      </div>
      <Property name="Mode" classes={{ content: classes.uppercaseContent }}>
        {props.mode}
      </Property>

      {props.status === 'refreshing' && props.refreshingStartDate && (
        <div className={classes.content}>
          <Property name="Refreshing started at">
            {formatUTC(props.refreshingStartDate, 'yyyy-MM-dd HH:mm:ss')} UTC (
            {formatDistanceToNowStrict(props.refreshingStartDate, {
              addSuffix: true,
            })}
            )
          </Property>
        </div>
      )}
      <div className={classes.content}>
        <Property name="Clones">{props.clonesCount}</Property>
        <Property name="Snapshots">{props.snapshotsCount}</Property>
      </div>

      <div className={classes.content}>
        <Property name="Size">{formatBytesIEC(props.totalDataSize)}</Property>
        <Property
          name={
            <>
              Used&nbsp;
              <Marker className={classes.markerUsed} />
            </>
          }
        >
          {formatBytesIEC(props.usedDataSize)} /{' '}
          {getPercent(props.usedDataSize, props.totalDataSize)} %
        </Property>
        <Property
          name={
            <>
              Free&nbsp;
              <Marker className={classes.markerFree} />
            </>
          }
        >
          {formatBytesIEC(props.freeDataSize)} /{' '}
          {getPercent(props.freeDataSize, props.totalDataSize)} %
        </Property>
      </div>

      <ProgressBar
        value={props.usedDataSize}
        total={props.totalDataSize}
        thresholdPercent={WARNING_THRESHOLD_PERCENT}
      />

      {shouldShowWarning && (
        <PerformanceStatus type="warning" className={classes.warningMessage}>
          +{WARNING_THRESHOLD_PERCENT}% disk usage may result in performance
          degradation
        </PerformanceStatus>
      )}
    </div>
  )
}
