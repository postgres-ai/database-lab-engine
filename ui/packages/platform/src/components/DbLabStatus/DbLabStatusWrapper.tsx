import { makeStyles } from '@material-ui/core'
import DbLabStatus, {
  DbLabStatusInstance,
} from 'components/DbLabStatus/DbLabStatus'
import { colors } from '@postgres.ai/shared/styles/colors'
import { Clone } from '@postgres.ai/shared/types/api/entities/clone'

export interface DbLabStatusProps {
  session?: { status: string }
  onlyText?: boolean
  showDescription?: boolean
  instance?: DbLabStatusInstance
  clone?: Clone
}

export const DbLabStatusWrapper = (props: DbLabStatusProps) => {
  const useStyles = makeStyles(
    {
      cloneReadyStatus: {
        color: colors.state.ok,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      cloneCreatingStatus: {
        color: colors.state.processing,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      cloneResettingStatus: {
        color: colors.state.processing,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      cloneDeletingStatus: {
        color: colors.state.warning,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      cloneFatalStatus: {
        color: colors.state.error,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      instanceReadyStatus: {
        color: colors.state.ok,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      instanceWarningStatus: {
        color: colors.state.warning,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      instanceNoResponseStatus: {
        color: colors.state.error,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      toolTip: {
        fontSize: '10px!important',
      },
      sessionPassedStatus: {
        display: 'inline-block',
        border: '1px solid ' + colors.state.ok,
        fontSize: '12px',
        color: '#FFFFFF',
        backgroundColor: colors.state.ok,
        padding: '3px',
        paddingLeft: '5px',
        paddingRight: '5px',
        borderRadius: 3,
        lineHeight: '14px',
        '& svg': {
          width: 10,
          height: 10,
          marginBottom: '-1px',
          marginRight: '5px',
        },
      },
      sessionFailedStatus: {
        display: 'inline-block',
        border: '1px solid ' + colors.state.error,
        fontSize: '12px',
        color: '#FFFFFF',
        backgroundColor: colors.state.error,
        padding: '3px',
        paddingLeft: '5px',
        paddingRight: '5px',
        borderRadius: 3,
        lineHeight: '14px',
        '& svg': {
          width: 10,
          height: 10,
          marginBottom: '-1px',
          marginRight: '5px',
        },
      },
      sessionProcessingStatus: {
        display: 'inline-block',
        border: '1px solid ' + colors.state.processing,
        fontSize: '12px',
        color: '#FFFFFF',
        backgroundColor: colors.state.processing,
        padding: '3px',
        paddingLeft: '5px',
        paddingRight: '5px',
        borderRadius: 3,
        lineHeight: '14px',
        '& svg': {
          width: 10,
          height: 10,
          marginBottom: '-1px',
          marginRight: '5px',
        },
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <DbLabStatus {...props} classes={classes} />
}
