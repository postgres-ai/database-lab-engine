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
import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import {
  TableBodyCell,
  TableBodyCellMenu,
  TableHeaderCell,
} from '@postgres.ai/shared/components/Table'

import { useCreatedStores } from './useCreatedStores'
import { Host } from './context'
import { DestroySnapshot } from '@postgres.ai/shared/types/api/endpoints/destroySnapshot'

type Props = Host

const useStyles = makeStyles(
  () => ({
    wrapper: {
      display: 'flex',
      gap: '60px',
      maxWidth: '1200px',
      fontSize: '14px',
      marginTop: '20px',

      '@media (max-width: 1300px)': {
        flexDirection: 'column',
        gap: '20px',
      },
    },
    marginTop: {
      marginTop: '16px',
    },
    container: {
      maxWidth: '100%',
      flex: '1 1 0',
      minWidth: 0,

      '&  p,span': {
        fontSize: 14,
      },
    },
    snippetContainer: {
      flex: '1 1 0',
      minWidth: 0,
      boxShadow: 'rgba(0, 0, 0, 0.1) 0px 4px 12px',
      padding: '10px 20px 10px 20px',
      height: 'max-content',
      borderRadius: '4px',
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
    cliText: {
      marginTop: '8px',
    },
    paramTitle: {
      display: 'inline-block',
      width: 200,
    },
    copyFieldContainer: {
      position: 'relative',
      display: 'block',
      maxWidth: 525,
      width: '100%',
    },
    tableContainer: {
      position: 'relative',
      maxWidth: 525,
      width: '100%',
      margin: '10px 0',
    },
    textField: {
      ...styles.inputField,
      'max-width': 525,
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
    centerContent: {
      display: 'flex',
      gap: 1,
      alignItems: 'center',
    },
    tableCellMenu: {
      width: 50,
      display: 'table-cell',
    },
  }),
  { index: 1 },
)

export const SnapshotPage = observer((props: Props) => {
  const classes = useStyles()
  const history = useHistory()
  const stores = useCreatedStores(props.api)

  const [isOpenDestroyModal, setIsOpenDestroyModal] = useState(false)

  const {
    snapshot,
    branchSnapshot,
    isSnapshotsLoading,
    snapshotError,
    branchSnapshotError,
    load,
  } = stores.main

  const afterSubmitClick = () => {
    history.push(props.routes.snapshots())
    load(props.snapshotId, props.instanceId)
  }

  const headRendered = (
    <>
      <style>{'p { margin: 0;}'}</style>

      {props.elements.breadcrumbs}

      <SectionTitle
        className={classes.marginTop}
        tag="h1"
        level={1}
        text={`Snapshot ${props.snapshotId}`}
      />
    </>
  )

  useEffect(() => {
    load(props.snapshotId, props.instanceId)
  }, [])

  if (isSnapshotsLoading) return <PageSpinner />

  if (snapshotError || branchSnapshotError) {
    return (
      <>
        {headRendered}
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
      {headRendered}
      <div className={classes.wrapper}>
        <div className={classes.container}>
          <div className={classes.actions}>
            <Button
              variant="contained"
              color="primary"
              onClick={() => history.push(props.routes.createClone(snapshot?.branch as string, snapshot?.id as string))}
              title={'Create clone'}
              className={classes.actionButton}
            >
              Create clone
            </Button>
            <Button
              variant="contained"
              color="primary"
              onClick={() => setIsOpenDestroyModal(true)}
              title={'Delete this snapshot'}
              className={classes.actionButton}
            >
              Delete snapshot
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
                <p className={classes.centerContent}>
                  <strong>
                    Related branches ({branchSnapshot.branch.length})
                  </strong>
                  &nbsp;
                  <Tooltip
                    content={
                      <>
                        List of branches pointing at the same snapshot. &nbsp;
                      </>
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
                      {branchSnapshot.branch.map(
                        (branch: string, id: number) => (
                          <TableRow
                            key={id}
                            className={classes.pointerCursor}
                            hover
                            onClick={() =>
                              history.push(props.routes.branch(branch))
                            }
                          >
                            <div className={classes.tableCellMenu}>
                              <TableBodyCellMenu
                                actions={[
                                  {
                                    name: 'Copy branch name',
                                    onClick: () => copyToClipboard(branch),
                                  },
                                ]}
                              />
                            </div>
                            <TableBodyCell>{branch}</TableBodyCell>
                          </TableRow>
                        ),
                      )}
                    </TableBody>
                  </Table>
                </HorizontalScrollContainer>
              </>
            )}
            <br />
            {snapshot?.clones && snapshot.clones.length > 0 && (
              <>
                <p>
                  <strong>Clones ({snapshot.clones.length})</strong>
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
                      {snapshot.clones.map((clone: string, id: number) => (
                        <TableRow
                          key={id}
                          className={classes.pointerCursor}
                          hover
                          onClick={() =>
                            history.push(props.routes.clone(clone))
                          }
                        >
                          <div className={classes.tableCellMenu}>
                            <TableBodyCellMenu
                              actions={[
                                {
                                  name: 'Copy clone name',
                                  onClick: () => copyToClipboard(clone),
                                },
                              ]}
                            />
                          </div>
                          <TableBodyCell>{clone}</TableBodyCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </HorizontalScrollContainer>
              </>
            )}
          </div>
        </div>
        <div className={classes.snippetContainer}>
          <SectionTitle
            className={classes.marginTop}
            tag="h2"
            level={2}
            text={'Delete snapshot using CLI'}
          />
          <p className={classes.cliText}>
            You can delete this snapshot using CLI. To do this, run the command
            below:
          </p>
          <SyntaxHighlight content={`dblab snapshot delete ${snapshot?.id}`} />
          <SectionTitle
            className={classes.marginTop}
            tag="h2"
            level={2}
            text={'Get snapshots using CLI'}
          />
          <p className={classes.cliText}>
            You can get a list of all snapshots using CLI. To do this, run the
            command below:
          </p>
          <SyntaxHighlight content={`dblab snapshot list`} />
        </div>
        {snapshot && (
          <DestroySnapshotModal
            isOpen={isOpenDestroyModal}
            onClose={() => setIsOpenDestroyModal(false)}
            snapshotId={snapshot.id}
            instanceId={props.instanceId}
            afterSubmitClick={afterSubmitClick}
            destroySnapshot={stores.main.destroySnapshot as DestroySnapshot}
          />
        )}
      </div>
    </>
  )
})
