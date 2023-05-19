import { Button } from '@material-ui/core'
import { AuditLogData } from 'components/Audit/Audit'

export const FilteredTableMessage = ({
  filterValue,
  filteredItems,
  clearFilter,
  emptyState,
}: {
  filterValue: string
  filteredItems: string[] | never[] | AuditLogData[] | undefined | null
  clearFilter: () => void
  emptyState: string | JSX.Element
}) => {
  if (filterValue && filteredItems?.length === 0) {
    return (
      <>
        <div>
          No results found for <b>{filterValue}</b>
        </div>
        <Button
          color="primary"
          variant="contained"
          onClick={clearFilter}
          style={{
            marginTop: 15,
            height: '33px',
            marginBottom: 10,
            maxWidth: 'max-content',
          }}
        >
          Clear filter
        </Button>
      </>
    )
  }

  return <>{emptyState}</>
}
