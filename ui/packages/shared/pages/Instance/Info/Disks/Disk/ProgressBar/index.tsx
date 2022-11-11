import { makeStyles } from '@material-ui/core'

import { colors } from '@postgres.ai/shared/styles/colors'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'

import pointerIconUrl from './pointer.svg'

type Props = {
  value: number
  total: number
  thresholdPercent: number
}

const useStyles = makeStyles(
  (theme) => ({
    '@keyframes grow': {
      '0%': {
        transform: 'scaleX(0)',
      },
      '100%': {
        transform: 'scaleX(1)',
      },
    },
    root: {
      height: '12px',
      position: 'relative',
      borderRadius: '4px',
      background: colors.gray,
      overflow: 'hidden',
      marginTop: '8px',
    },
    indicator: {
      position: 'absolute',
      top: 0,
      left: 0,
      height: '100%',
      width: '30%',
      background: colors.primary.light,
      animation: `$grow 500ms ${theme.transitions.easing.easeOut}`,
      transformOrigin: 0,
    },
    pointer: {
      position: 'absolute',
      height: '100%',
      top: 0,
      transform: 'translateX(-50%)',
    },
  }),
  { index: 1 },
)

export const ProgressBar = (props: Props) => {
  const classes = useStyles()

  return (
    <div className={classes.root}>
      <div
        className={classes.indicator}
        style={{ width: `${(props.value / props.total) * 100}%` }}
      />
      <Tooltip
        content={`+${props.thresholdPercent}% disk usage may result in performance degradation`}
      >
        <img
          src={pointerIconUrl}
          alt="pointer"
          className={classes.pointer}
          style={{
            left: `${props.thresholdPercent}%`,
          }}
        />
      </Tooltip>
    </div>
  )
}
