/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useEffect, useState } from 'react'
import { useHistory } from 'react-router'
import { observer } from 'mobx-react-lite'
import copyToClipboard from 'copy-to-clipboard'
import {
  makeStyles,
  Button,
  TextField,
  IconButton,
  Table,
  TableHead,
  TableRow,
  TableBody,
} from '@material-ui/core'

import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { DestroySnapshotModal } from '@postgres.ai/shared/pages/Snapshots/Snapshot/DestorySnapshotModal'
import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { icons } from '@postgres.ai/shared/styles/icons'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { styles } from '@postgres.ai/shared/styles/styles'
import {
  TableBodyCell,
  TableBodyCellMenu,
  TableHeaderCell,
} from '@postgres.ai/shared/components/Table'

import { useCreatedStores } from './useCreatedStores'
import { Host } from './context'

type Props = Host

const useStyles = makeStyles(
  () => ({
    marginTop: {
      marginTop: '16px',
    },
    container: {
      maxWidth: '100%',
      marginTop: '16px',

      '&  p,span': {
        fontSize: 14,
      },
    },
    actions: {
      display: 'flex',
      marginRight: '-16px',
    },
    spinner: {
      marginLeft: '8px',
    },
    actionButton: {
      marginRight: '16px',
    },
    summary: {
      marginTop: 20,
    },
    text: {
      marginTop: '4px',
    },
    paramTitle: {
      display: 'inline-block',
      width: 200,
    },
    copyFieldContainer: {
      position: 'relative',
      display: 'block',
      maxWidth: 400,
      width: '100%',
    },
    tableContainer: {
      position: 'relative',
      maxWidth: 400,
      width: '100%',
    },
    textField: {
      ...styles.inputField,
      'max-width': 400,
      display: 'inline-block',
      '& .MuiOutlinedInput-input': {
        paddingRight: '32px!important',
      },
    },
    copyButton: {
      position: 'absolute',
      top: 16,
      right: 0,
      zIndex: 100,
      width: 32,
      height: 32,
      padding: 8,
    },
    pointerCursor: {
      cursor: 'pointer',
    },
  }),
  { index: 1 },
)

export const SnapshotPage = observer((props: Props) => {
  const classes = useStyles()
  const history = useHistory()
  const stores = useCreatedStores(props)

  const [isOpenDestroyModal, setIsOpenDestroyModal] = useState(false)

  const {
    snapshot,
    branchSnapshot,
    isSnapshotsLoading,
    snapshotError,
    branchSnapshotError,
    destroySnapshotError,
    load,
  } = stores.main

  const destroySnapshot = async () => {
    const isSuccess = await stores.main.destroySnapshot(String(snapshot?.id))
    if (isSuccess) history.push(props.routes.snapshot())
  }

  const BranchHeader = () => {
    return (
      <>
        {props.elements.breadcrumbs}
        <SectionTitle
          className={classes.marginTop}
          tag="h1"
          level={1}
          text={`Snapshot ${props.snapshotId}`}
        />
      </>
    )
  }

  useEffect(() => {
    load(props.snapshotId, props.instanceId)
  }, [])

  if (isSnapshotsLoading) return <PageSpinner />

  if (snapshotError || branchSnapshotError) {
    return (
      <>
        <BranchHeader />
        <ErrorStub
          title={snapshotError?.title || branchSnapshotError?.title}
          message={snapshotError?.message || branchSnapshotError?.message}
          className={classes.marginTop}
        />
      </>
    )
  }

  return (
    <>
      <BranchHeader />
      <div className={classes.container}>
        <div className={classes.actions}>
          <Button
            variant="contained"
            color="primary"
            onClick={() => setIsOpenDestroyModal(true)}
            title={'Destroy this snapshot'}
            className={classes.actionButton}
          >
            Destroy snapshot
          </Button>
        </div>
        <br />
        <div>
          <div>
            <p>
              <strong>Created</strong>
            </p>
            <p className={classes.text}>{snapshot?.createdAt}</p>
          </div>
          <br />
          <div>
            <p>
              <strong>Data state at</strong>&nbsp;
              <Tooltip
                content={
                  <>
                    <strong>Data state time</strong> is a time at which data
                    is&nbsp; recovered for this snapshot.
                  </>
                }
              >
                {icons.infoIcon}
              </Tooltip>
            </p>
            <p className={classes.text}>{snapshot?.dataStateAt || '-'}</p>
          </div>
          <div className={classes.summary}>
            <p>
              <strong>Summary</strong>&nbsp;
            </p>
            <p className={classes.text}>
              <span className={classes.paramTitle}>Number of clones:</span>
              {snapshot?.numClones}
            </p>
            <p className={classes.text}>
              <span className={classes.paramTitle}>Logical data size:</span>
              {snapshot?.logicalSize
                ? formatBytesIEC(snapshot.logicalSize)
                : '-'}
            </p>
            <p className={classes.text}>
              <span className={classes.paramTitle}>
                Physical data diff size:
              </span>
              {snapshot?.physicalSize
                ? formatBytesIEC(snapshot.physicalSize)
                : '-'}
            </p>
            {branchSnapshot?.message && (
              <p className={classes.text}>
                <span className={classes.paramTitle}>Message:</span>
                {branchSnapshot.message}
              </p>
            )}
          </div>
          <br />
          <p>
            <strong>Snapshot info</strong>
          </p>
          {snapshot?.pool && (
            <div className={classes.copyFieldContainer}>
              <TextField
                variant="outlined"
                label="snapshot pool"
                value={snapshot.pool}
                className={classes.textField}
                margin="normal"
                fullWidth
                InputLabelProps={{
                  shrink: true,
                  style: styles.inputFieldLabel,
                }}
                FormHelperTextProps={{
                  style: styles.inputFieldHelper,
                }}
              />
              <IconButton
                className={classes.copyButton}
                aria-label="Copy"
                onClick={() => copyToClipboard(snapshot.pool)}
              >
                {icons.copyIcon}
              </IconButton>
            </div>
          )}
          <div className={classes.copyFieldContainer}>
            <TextField
              variant="outlined"
              label="snapshot ID"
              value={snapshot?.id}
              className={classes.textField}
              margin="normal"
              fullWidth
              InputLabelProps={{
                shrink: true,
                style: styles.inputFieldLabel,
              }}
              FormHelperTextProps={{
                style: styles.inputFieldHelper,
              }}
            />
            <IconButton
              className={classes.copyButton}
              aria-label="Copy"
              onClick={() => copyToClipboard(String(snapshot?.id))}
            >
              {icons.copyIcon}
            </IconButton>
          </div>
          <br />
          {branchSnapshot?.branch && branchSnapshot.branch?.length > 0 && (
            <>
              <p>
                <strong>
                  Related branches ({branchSnapshot.branch.length})
                </strong>
                &nbsp;
                <Tooltip
                  content={
                    <>List of branches pointing at the same snapshot. &nbsp;</>
                  }
                >
                  {icons.infoIcon}
                </Tooltip>
              </p>
              <HorizontalScrollContainer>
                <Table className={classes.tableContainer}>
                  <TableHead>
                    <TableRow>
                      <TableHeaderCell />
                      <TableHeaderCell>Name</TableHeaderCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {branchSnapshot.branch.map((branch: string, id: number) => (
                      <TableRow
                        key={id}
                        className={classes.pointerCursor}
                        hover
                        onClick={() =>
                          history.push(`/instance/branches/${branch}`)
                        }
                      >
                        <TableBodyCellMenu
                          actions={[
                            {
                              name: 'Copy branch name',
                              onClick: () => copyToClipboard(branch),
                            },
                          ]}
                        />
                        <TableBodyCell>{branch}</TableBodyCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </HorizontalScrollContainer>
            </>
          )}
        </div>
        <DestroySnapshotModal
          isOpen={isOpenDestroyModal}
          onClose={() => setIsOpenDestroyModal(false)}
          snapshotId={props.snapshotId}
          onDestroySnapshot={destroySnapshot}
          destroySnapshotError={destroySnapshotError}
        />
      </div>
    </>
  )
})
