import { observer } from 'mobx-react-lite'
import { makeStyles, TextField } from '@material-ui/core'
import { Select } from '@postgres.ai/shared/components/Select'

interface SnapshotHeaderProps {
  branches: string[] | null
  selectedBranch: string
  setMessageFilter: (value: string) => void
  setSelectedBranch: (value: string) => void
}

const useStyles = makeStyles(
  {
    outerContainer: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      paddingBottom: '6px',
    },
    select: {
      width: '200px',
    },
    inputContainer: {
      width: '300px',

      '& input': {
        padding: '8px',
      },
    },
  },
  { index: 1 },
)

export const SnapshotHeader = observer(
  ({
    branches,
    selectedBranch,
    setMessageFilter,
    setSelectedBranch,
  }: SnapshotHeaderProps) => {
    const classes = useStyles()

    return (
      <div className={classes.outerContainer}>
        <Select
          fullWidth
          label="Branch"
          className={classes.select}
          value={selectedBranch}
          disabled={!branches}
          onChange={(e) => {
            setSelectedBranch(e.target.value)
          }}
          items={
            branches
              ? branches.map((branch) => {
                  return {
                    value: branch,
                    children: <div>{branch}</div>,
                  }
                })
              : []
          }
        />
        <TextField
          variant="outlined"
          className={classes.inputContainer}
          onChange={(e) => setMessageFilter(e.target.value)}
          label={'Filter by message'}
          InputLabelProps={{
            shrink: true,
          }}
        />
      </div>
    )
  },
)
