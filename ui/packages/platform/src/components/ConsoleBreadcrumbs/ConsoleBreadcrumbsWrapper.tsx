import { makeStyles } from '@material-ui/core'
import { colors } from '@postgres.ai/shared/styles/colors'
import ConsoleBreadcrumbs from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbs'

export interface ConsoleBreadcrumbsProps {
  hasDivider?: boolean
  org?: string | number
  project?: string
  breadcrumbs: { name: string; url?: string | null; isLast?: boolean }[]
}

export const ConsoleBreadcrumbsWrapper = (props: ConsoleBreadcrumbsProps) => {
  const useStyles = makeStyles(
    {
      pointerLink: {
        cursor: 'pointer',
      },
      breadcrumbsLink: {
        maxWidth: 150,
        textOverflow: 'ellipsis',
        overflow: 'hidden',
        display: 'block',
        cursor: 'pointer',
        whiteSpace: 'nowrap',
        fontSize: '12px',
        lineHeight: '14px',
        textDecoration: 'none',
        color: colors.consoleFadedFont,
      },
      breadcrumbsItem: {
        fontSize: '12px',
        lineHeight: '14px',
        color: colors.consoleFadedFont,
      },
      breadcrumbsActiveItem: {
        fontSize: '12px',
        lineHeight: '14px',
        color: '#000000',
      },
      breadcrumbPaper: {
        '& a, & a:visited': {
          color: colors.consoleFadedFont,
        },
        'padding-bottom': '8px',
        marginTop: '-10px',
        'font-size': '12px',
        borderRadius: 0,
      },
      breadcrumbPaperWithDivider: {
        borderBottom: `1px solid ${colors.consoleStroke}`,
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <ConsoleBreadcrumbs {...props} classes={classes} />
}
