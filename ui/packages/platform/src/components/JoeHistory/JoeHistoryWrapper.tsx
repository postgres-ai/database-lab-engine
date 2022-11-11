import { makeStyles } from '@material-ui/core'
import JoeHistory from 'components/JoeHistory/JoeHistory'
import { styles } from '@postgres.ai/shared/styles/styles'
import { colors } from '@postgres.ai/shared/styles/colors'
import { QueryParamsType } from 'components/types'
import { RouteComponentProps } from 'react-router'

export interface JoeHistoryProps extends QueryParamsType {
  auth: {
    token: string
  } | null
  orgId: number
  org: string | number
  userIsOwner: boolean
  history: RouteComponentProps['history']
}

export const JoeHistoryWrapper = (props: JoeHistoryProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        ...(styles.root as Object),
        display: 'flex',
        flexDirection: 'column',
        paddingBottom: '20px',
      },
      filterSelect: {
        ...styles.inputField,
        width: 150,
        marginRight: '15px',
      },
      searchFilter: {
        ...styles.inputField,
        '& input.MuiInputBase-input': {
          paddingLeft: '0px!important',
        },
      },
      checkboxTableCell: {
        width: '30px',
        padding: 0,
        paddingTop: 10,
        verticalAlign: 'top',
      },
      tag: {
        border: '1px solid ' + colors.consoleStroke,
        borderRadius: 3,
        fontSize: 10,
        lineHeight: '16px',
        display: 'inline-block',
        marginRight: 8,
        backgroundColor: 'white',
        '&:last-child': {
          marginRight: 0,
        },
      },
      tagName: {
        fontSize: 10,
        color: colors.pgaiDarkGray,
        borderRight: '1px solid ' + colors.consoleStroke,
        paddingLeft: 3,
        paddingRight: 3,
        backgroundColor: 'white',
      },
      tagValue: {
        fontSize: 10,
        color: colors.secondary2.main,
        paddingLeft: 3,
        paddingRight: 3,
        backgroundColor: 'white',
      },
      twoSideRow: {
        display: 'flex',
      },
      twoSideCol1: {},
      twoSideCol2: {
        flexGrow: 1,
        textAlign: 'right',
      },
      timeLabel: {
        lineHeight: '16px',
        fontSize: 12,
        color: colors.pgaiDarkGray,
      },
      query: {
        marginTop: 10,
        marginBottom: 10,
        fontSize: 14,
        lineHeight: '16px',
        fontFamily:
          '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
          ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
        color: 'black',
        '& > b': {
          color: colors.secondary2.main,
        },
        height: 32,
        overflow: 'hidden',
        'text-overflow': 'ellipsis',
        '-webkit-line-clamp': '2',
        '-webkit-box-orient': 'vertical',
        display: '-webkit-box',
      },
      showMoreContainer: {
        marginTop: 20,
        textAlign: 'center',
      },
      cardCell: {
        paddingLeft: 0,
        paddingRight: 0,
      },
      toolTip: {
        fontSize: '10px!important',
        maxWidth: '420px!important',
      },
      searchIcon: {
        marginRight: 0,
      },
      filterContainer: {
        display: 'flex',
        [theme.breakpoints.down('sm')]: {
          display: 'block',
        },
      },
      filterButton: {
        height: '32px',
        marginRight: 10,
        marginBottom: 10,
      },
      tableHead: {
        ...(styles.tableHead as Object),
        fontWeight: 'normal',
        padding: 0,
        paddingBottom: 5,
        paddingTop: 5,
      },
      tableHeadActions: {
        ...(styles.tableHeadActions as Object),
      },
      headCheckboxTableCell: {
        width: '30px',
        padding: 0,
        verticalAlign: 'top',
        paddingBottom: 5,
        paddingTop: 5,
      },
      centeredBox: {
        display: 'flex',
        justifyContent: 'center',
        height: 'calc(100vh - 250px)',
        mixHeight: '220px',
      },
      centeredProgress: {
        marginTop: 'calc((100vh - 250px) / 2 - 30px)',
      },
      whiteSpace: {
        whiteSpace: 'nowrap',
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <JoeHistory {...props} classes={classes} />
}
