import { makeStyles } from '@material-ui/core'
import Warning from 'components/Warning/Warning'
import { colors } from '@postgres.ai/shared/styles/colors'

export interface WarningProps {
  children: React.ReactNode
  inline?: boolean
  actions?: {
    id: number
    content: string
  }[]
}

export const WarningWrapper = (props: WarningProps) => {
  const linkColor = colors.secondary2.main

  const useStyles = makeStyles(
    {
      root: {
        backgroundColor: colors.secondary1.lightLight,
        color: colors.secondary1.darkDark,
        fontSize: '14px',
        paddingTop: '5px',
        paddingBottom: '5px',
        paddingLeft: '10px',
        paddingRight: '10px',
        display: 'flex',
        alignItems: 'center',
        borderRadius: '3px',
      },
      block: {
        marginBottom: '20px',
      },
      icon: {
        padding: '5px',
      },
      actions: {
        '& a': {
          color: linkColor,
          '&:visited': {
            color: linkColor,
          },
          '&:hover': {
            color: linkColor,
          },
          '&:active': {
            color: linkColor,
          },
        },
        marginRight: '15px',
      },
      container: {
        marginLeft: '5px',
        marginRight: '5px',
        lineHeight: '24px',
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <Warning {...props} classes={classes} />
}
