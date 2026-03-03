import { makeStyles } from '@material-ui/core'
import { formatDistanceToNowStrict } from 'date-fns'

import { colors } from '@postgres.ai/shared/styles/colors'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { formatUTC, isValidDate } from '@postgres.ai/shared/utils/date'

import { Property } from '../../../components/Property'
import { ActionsMenu } from '../../Disk/ActionsMenu'
import { Status } from '../../Disk/Status'

export type DatasetInfo = {
  id: string | null
  name: string
  showName: boolean
  status: 'refreshing' | 'active' | 'empty'
  mode: string
  usedDataSize: number
  clonesCount: number
  snapshotsCount: number
  refreshingStartDate: Date | null
}

const useStyles = makeStyles(
  {
    root: {
      border: `1px solid ${colors.consoleStroke}`,
      padding: '6px 8px 8px',
      borderRadius: '4px',
      backgroundColor: colors.white,
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
    uppercaseContent: {
      textTransform: 'uppercase',
    },
  },
  { index: 1 },
)

export const DatasetRow = (props: DatasetInfo) => {
  const classes = useStyles()

  return (
    <div className={classes.root}>
      <div className={classes.header}>
        {props.showName ? (
          <div className={classes.titleWrapper}>
            <h6 title={props.name} className={classes.title}>
              {props.name}
            </h6>
            <ActionsMenu
              poolId={props.id}
              poolName={props.id ?? props.name}
              isActive={props.status === 'active'}
            />
          </div>
        ) : (
          <ActionsMenu
            poolId={props.id}
            poolName={props.id ?? props.name}
            isActive={props.status === 'active'}
          />
        )}
        <Status value={props.status} hasWarning={false} />
      </div>

      <Property name="Mode" classes={{ content: classes.uppercaseContent }}>
        {props.mode}
      </Property>

      {props.status === 'refreshing' && props.refreshingStartDate && (
        <div className={classes.content}>
          <Property name="Refreshing started at">
            {formatUTC(props.refreshingStartDate, 'yyyy-MM-dd HH:mm:ss')} UTC (
            {isValidDate(props.refreshingStartDate)
              ? formatDistanceToNowStrict(props.refreshingStartDate, {
                  addSuffix: true,
                })
              : '-'}
            )
          </Property>
        </div>
      )}

      <div className={classes.content}>
        <Property name="Clones">{props.clonesCount}</Property>
        <Property name="Snapshots">{props.snapshotsCount}</Property>
      </div>

      <div className={classes.content}>
        <Property name="Size">{formatBytesIEC(props.usedDataSize)}</Property>
      </div>
    </div>
  )
}
