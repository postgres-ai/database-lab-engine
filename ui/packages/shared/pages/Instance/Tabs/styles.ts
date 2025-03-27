import { makeStyles } from '@material-ui/core'
import { colors } from '@postgres.ai/shared/styles/colors'

export const useTabsStyles = makeStyles({
  tabsRoot: {
    minHeight: 0,
    marginTop: '-8px',
    '& .MuiTabs-fixed': {
      overflowX: 'auto!important',
    },
    '& .postgres-logo': {
      width: '18px',
      height: '18px',
    },
    '& a': {
      color: colors.black,
      textDecoration: 'none',
    },
  },
  flexRow: {
    display: 'flex',
    flexDirection: 'row',
    gap: '5px',
  },
  tabsIndicator: {
    height: '3px',
  },
  tabRoot: {
    fontWeight: 400,
    minWidth: 0,
    minHeight: 0,
    padding: '6px 16px',
    borderBottom: `3px solid ${colors.consoleStroke}`,
    '& + $tabRoot': {
      marginLeft: '10px',
    },
    '&.Mui-disabled': {
      opacity: 1,
      color: colors.pgaiDarkGray,
    },
  },
  tabHidden: {
    display: 'none',
  },
}, { index: 1 })
