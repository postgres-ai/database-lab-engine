import React from 'react'
import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'
import copy from 'copy-to-clipboard'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { DestroySnapshotModal } from '@postgres.ai/shared/pages/Snapshots/Snapshot/DestorySnapshotModal'
import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { IconButton } from '@mui/material'
import { icons } from '@postgres.ai/shared/styles/icons'
import { RowMenu } from '@postgres.ai/shared/components/Table/RowMenu'
import {
  generateSnapshotPageId,
  groupSnapshotsByCreatedAtDate,
} from '@postgres.ai/shared/pages/Instance/Snapshots/utils'
import { format, formatDistanceToNowStrict } from 'date-fns'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { useHistory } from 'react-router'
import { DestroySnapshot } from '@postgres.ai/shared/types/api/endpoints/destroySnapshot'

const useStyles = makeStyles(
  {
    pointerCursor: {
      cursor: 'pointer',
      padding: '0 8px',

      '&:hover': {
        backgroundColor: '#f9f9f9',
      },
    },
    commitItem: {
      display: 'flex',
      justifyContent: 'space-between',
      padding: '14px 0',
      borderBottom: '1px solid #ddd',

      '&:last-child': {
        borderBottom: 'none',
        marginBottom: '12px',
      },
    },
    infoBlock: {
      display: 'flex',
      flexDirection: 'column',
      justifyContent: 'center',
      overflow: 'hidden',
    },
    header: {
      fontWeight: 500,
    },
    infoContent: {
      fontSize: '12px',
      color: '#808080',
      display: 'flex',
      alignItems: 'center',
      gap: '5px',
    },
    snapshotId: {
      fontSize: '12px',
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
      padding: '0 10px',
    },
    actionsContainer: {
      overflow: 'hidden',
      display: 'flex',
      height: 'max-content',
      alignItems: 'center',
      justifyContent: 'flex-end',
      border: '1px solid #ddd',
      borderRadius: '5px',
      padding: '0 14px',
      justifySelf: 'end',
      maxWidth: '100%',

      '& .MuiButtonBase-root, .MuiTableCell-root': {
        padding: '0!important',
        borderBottom: 'none',
      },
    },
    dateGroup: {
      fontSize: '14px',
      fontWeight: 'bold',
      padding: '14px 0 5px 0',
      borderBottom: '1px solid #ddd',
    },
    rowMenuContainer: {
      paddingLeft: '10px',
      '& .MuiButtonBase-root': {
        width: '20px',
        height: '20px',
        padding: '8px',
      },
    },
    copyButtonContainer: {
      '& .MuiButtonBase-root': {
        borderLeft: '1px solid #ddd',
        borderRadius: '0',
        borderRight: '1px solid #ddd',
      },
    },
    copyButton: {
      width: '32px',
      height: '32px',
      padding: '8px',
    },
    tooltipInfo: {
      fontSize: '12px',
    },
    gridContainer: {
      width: '100%',
      display: 'grid',
      columnGap: '20px',
      gridTemplateColumns: 'repeat(5, 0.75fr) 1fr',
    },
  },
  { index: 1 },
)

const SnapshotListItem = ({
  snapshot,
  setSnapshotModal,
  openClonesModal,
}: {
  snapshot: Snapshot
  openClonesModal: () => void
  setSnapshotModal: (modal: { isOpen: boolean; snapshotId: string }) => void
}) => {
  const classes = useStyles()
  const timeAgo = formatDistanceToNowStrict(snapshot.createdAtDate)

  return (
    <div className={classes.commitItem}>
      <div className={classes.gridContainer}>
        <div className={classes.infoBlock}>
          <div className={classes.header}>{snapshot.message || '-'}</div>
          <div className={classes.infoContent} title={snapshot.dataStateAt}>
            {timeAgo} ago
          </div>
        </div>
        <div className={classes.infoBlock}>
          <div className={classes.header}>Pool</div>
          <div className={classes.infoContent}>{snapshot.pool ?? '-'}</div>
        </div>
        <div className={classes.infoBlock}>
          <div className={classes.header}>Number of clones</div>
          <div className={classes.infoContent}>{snapshot.numClones ?? '-'}</div>
        </div>
        <div className={classes.infoBlock}>
          <div className={classes.header}>Logical Size</div>
          <div className={classes.infoContent}>
            {snapshot.logicalSize ? formatBytesIEC(snapshot.logicalSize) : '-'}
          </div>
        </div>
        <div className={classes.infoBlock}>
          <div className={classes.header}>Physical Size</div>
          <div className={classes.infoContent}>
            {snapshot.physicalSize
              ? formatBytesIEC(snapshot.physicalSize)
              : '-'}
          </div>
        </div>
        <div
          className={classes.actionsContainer}
          onClick={(e) => e.stopPropagation()}
        >
          <div className={classes.snapshotId}>{snapshot.id}</div>
          <div className={classes.copyButtonContainer} title="Copy snapshot ID">
            <IconButton
              className={classes.copyButton}
              onClick={(e) => {
                e.stopPropagation()
                copy(snapshot.id)
              }}
            >
              {icons.copyIcon}
            </IconButton>
          </div>
          <div className={classes.rowMenuContainer} title="Actions">
            <RowMenu
              actions={[
                {
                  name: 'Show related clones',
                  onClick: () => openClonesModal(),
                },
                {
                  name: 'Delete snapshot',
                  onClick: () =>
                    setSnapshotModal({
                      isOpen: true,
                      snapshotId: snapshot.id,
                    }),
                },
              ]}
            />
          </div>
        </div>
      </div>
    </div>
  )
}

export const SnapshotsList = observer(
  ({
    routes,
    filteredSnapshots,
    instanceId,
  }: {
    routes: {
      snapshot: (snapshotId: string) => string
    }
    filteredSnapshots: Snapshot[]
    instanceId: string
  }) => {
    const classes = useStyles()
    const stores = useStores()
    const history = useHistory()
    const groupedSnapshots = groupSnapshotsByCreatedAtDate(filteredSnapshots)

    const [snapshotModal, setSnapshotModal] = React.useState({
      isOpen: false,
      snapshotId: '',
    })

    return (
      <HorizontalScrollContainer>
        {groupedSnapshots.map((group, index) => {
          const groupDateFormatted = format(
            group[0].createdAtDate,
            'MMM dd, yyyy',
          )

          return (
            <div key={index}>
              <div className={classes.dateGroup}>{groupDateFormatted}</div>
              {group.map((snapshot) => {
                const snapshotPageId = generateSnapshotPageId(snapshot.id)
                return (
                  <div
                    key={snapshot.id}
                    className={classes.pointerCursor}
                    onClick={() =>
                      snapshotPageId &&
                      history.push(routes.snapshot(snapshotPageId))
                    }
                  >
                    <SnapshotListItem
                      snapshot={snapshot}
                      setSnapshotModal={setSnapshotModal}
                      openClonesModal={() =>
                        stores.clonesModal.openModal({
                          snapshotId: snapshot.id,
                        })
                      }
                    />
                  </div>
                )
              })}
            </div>
          )
        })}
        {snapshotModal.isOpen && snapshotModal.snapshotId && (
          <DestroySnapshotModal
            isOpen={snapshotModal.isOpen}
            onClose={() => setSnapshotModal({ isOpen: false, snapshotId: '' })}
            snapshotId={snapshotModal.snapshotId}
            instanceId={instanceId}
            afterSubmitClick={() => stores.main.load(instanceId)}
            destroySnapshot={stores.main.destroySnapshot as DestroySnapshot}
          />
        )}
      </HorizontalScrollContainer>
    )
  },
)
