import { makeStyles, Theme } from '@material-ui/core'

export const useTabsStyles = makeStyles(
  (theme: Theme) => ({
    tabsRoot: {
      minHeight: 0,
      marginTop: '-8px',
      '& .MuiTabs-fixed': {
        overflowX: 'auto!important',
        scrollbarWidth: 'none',
        '&::-webkit-scrollbar': {
          display: 'none',
        },

        '& div:first-child': {
          '@media (max-width: 700px)': {
            display: 'flex',
          },
        },
      },
      '& .postgres-logo': {
        width: '18px',
        height: '18px',
      },
      '& a': {
        color: theme.palette.text.primary,
        textDecoration: 'none',

        '@media (max-width: 700px)': {
          display: 'flex',
          width: '100%',
        },
      },
    },
    flexRow: {
      display: 'flex',
      flexDirection: 'row',
      gap: '5px',
      width: '100%',
    },
    tabsIndicator: {
      height: '3px',
    },
    tabRoot: {
      color: theme.palette.text.primary,
      fontWeight: 400,
      minWidth: 0,
      minHeight: 0,
      width: '100%',
      padding: '6px 16px',
      borderBottom: `3px solid ${theme.palette.divider}`,
      '& + $tabRoot': {
        marginLeft: '10px',
      },
      '&.Mui-disabled': {
        opacity: 1,
        color: theme.palette.text.disabled,
      },
      '@media (max-width: 700px)': {
        width: 'max-content',
      },
    },
    tabHidden: {
      display: 'none',
    },
  }),
  { index: 1 },
)
