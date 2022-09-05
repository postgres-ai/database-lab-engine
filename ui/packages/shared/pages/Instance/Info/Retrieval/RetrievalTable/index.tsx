import {
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
} from '@material-ui/core'
import { ActivityType } from 'types/api/entities/instanceRetrieval'

import styles from './styles.module.scss'

export const RetrievalTable = ({
  data,
  activity,
}: {
  data: ActivityType[]
  activity: string
}) => {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>Activity on the {activity}</TableCell>
        </TableRow>
      </TableHead>
      <TableBody className={styles.tableBody}>
        {data && data.length > 0 ? (
          data.map((item) => (
            <div>
              {Object.entries(item).map((val, index) => (
                <TableRow key={index} hover className={styles.tableRow}>
                  <TableCell>
                    {val[0]} : {val[1]}
                  </TableCell>
                </TableRow>
              ))}
            </div>
          ))
        ) : (
          <TableRow className={styles.tableRow}>
            <TableCell>No activity on the {activity}</TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  )
}
