import { makeStyles } from '@material-ui/core'
import ConsoleButton from 'components/ConsoleButton/ConsoleButton'

export interface ConsoleButtonProps {
  title: string
  children: React.ReactNode
  className?: string
  disabled?: boolean
  variant: 'text' | 'outlined' | 'contained' | undefined
  color: 'primary' | 'secondary' | undefined
  onClick: () => void
  id?: string
}

export const ConsoleButtonWrapper = (props: ConsoleButtonProps) => {
  const useStyles = makeStyles(
    {
      tooltip: {
        fontSize: '10px!important',
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <ConsoleButton {...props} classes={classes} />
}
