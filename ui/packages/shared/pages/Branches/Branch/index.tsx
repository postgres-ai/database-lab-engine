/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useEffect, useState } from 'react'
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
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { icons } from '@postgres.ai/shared/styles/icons'
import { styles } from '@postgres.ai/shared/styles/styles'
import { DeleteBranchModal } from '@postgres.ai/shared/pages/Branches/components/Modals/DeleteBranchModal'
import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { generateSnapshotPageId } from '@postgres.ai/shared/pages/Instance/Snapshots/utils'
import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import { getCliBranchListCommand } from '@postgres.ai/shared/pages/CreateBranch/utils'
import {
  TableBodyCell,
  TableBodyCellMenu,
  TableHeaderCell,
} from '@postgres.ai/shared/components/Table'

import { useCreatedStores } from './useCreatedStores'
import { Host } from './context'
import { DeleteBranch } from '@postgres.ai/shared/types/api/endpoints/deleteBranch'
import { InstanceTabs, TABS_INDEX } from "../../Instance/Tabs";

type Props = Host & { isPlatform?: boolean, hideBranchingFeatures?: boolean }

const useStyles = makeStyles(
  () => ({
    wrapper: {
      display: 'flex',
      gap: '60px',
      maxWidth: '100%',
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
    title: {
      marginTop: '8px',
      lineHeight: '26px'
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
    textField: {
      ...styles.inputField,
      'max-width': 525,
      display: 'inline-block',
      '& .MuiOutlinedInput-input': {
        paddingRight: '32px!important',
      },
    },
    tableContainer: {
      position: 'relative',
      maxWidth: 525,
      width: '100%',
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

export const BranchesPage = observer((props: Props) => {
  const classes = useStyles()
  const history = useHistory()
  const stores = useCreatedStores(props)

  const [isOpenDestroyModal, setIsOpenDestroyModal] = useState(false)

  const {
    branch,
    snapshotList,
    deleteBranch,
    reload,
    load,
    isReloading,
    isBranchesLoading,
    getBranchesError,
    snapshotListError,
    getBranchError,
  } = stores.main

  const hasBranchError = getBranchesError || getBranchError || snapshotListError

  const branchLogLength = snapshotList?.reduce((acc, snapshot) => {
    if (snapshot?.branch !== null) {
      return acc + snapshot.branch?.length
    } else {
      return acc
    }
  }, 0)

  const headRendered = (
    <>
      <style>{'p { margin: 0;}'}</style>

      {props.elements.breadcrumbs}

      <SectionTitle
        className={classes.title}
        tag="h1"
        level={1}
        text={`Branch ${props.branchId}`}
      >
        <InstanceTabs
          tab={TABS_INDEX.BRANCHES}
          isPlatform={props.isPlatform}
          instanceId={props.instanceId}
          hasLogs={props.api.initWS !== undefined}
          hideInstanceTabs={props.hideBranchingFeatures}
        />
      </SectionTitle>
    </>
  )

  useEffect(() => {
    load(props.branchId, props.instanceId)
  }, [])

  if (isBranchesLoading) return <PageSpinner />

  if (hasBranchError) {
    return (
      <>
        {headRendered}
        <ErrorStub
          title={
            getBranchesError?.title ||
            getBranchError?.title ||
            snapshotListError?.title
          }
          message={
            getBranchesError?.message ||
            getBranchError?.message ||
            snapshotListError?.message
          }
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
              onClick={() => history.push(props.routes.createClone(props.branchId))}
              disabled={isReloading}
              title={'Create clone'}
              className={classes.actionButton}
            >
              Create clone
            </Button>
            <Button
              variant="contained"
              color="primary"
              onClick={() => setIsOpenDestroyModal(true)}
              disabled={isReloading}
              title={'Delete this branch'}
              className={classes.actionButton}
            >
              Delete branch
            </Button>
            <Button
              variant="outlined"
              color="secondary"
              onClick={() => reload(props.branchId, props.instanceId)}
              disabled={isReloading}
              title={'Refresh branch information'}
              className={classes.actionButton}
            >
              Reload info
              {isReloading && <Spinner size="sm" className={classes.spinner} />}
            </Button>
          </div>
          <br />
          <div>
            <div>
              <p>
                <strong>Data state at</strong>&nbsp;
                <Tooltip
                  content={
                    <>
                      <strong>Data state time</strong> is a time at which data
                      is&nbsp; recovered for this branch.
                    </>
                  }
                >
                  {icons.infoIcon}
                </Tooltip>
              </p>
              <p className={classes.text}>{branch?.dataStateAt || '-'}</p>
            </div>
            <div className={classes.summary}>
              <p>
                <strong>Summary</strong>&nbsp;
              </p>
              <p className={classes.text}>
                <span className={classes.paramTitle}>Branch name:</span>
                {branch?.name}
              </p>
              <p className={classes.text}>
                <span className={classes.paramTitle}>Parent branch:</span>
                {branch?.parent}
              </p>
            </div>
            <br />
            <p>
              <strong>Snapshot info</strong>
            </p>
            <div className={classes.copyFieldContainer}>
              <TextField
                variant="outlined"
                label="Snapshot ID"
                value={branch?.snapshotID}
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
                onClick={() => copyToClipboard(String(branch?.snapshotID))}
              >
                {icons.copyIcon}
              </IconButton>
            </div>
            <br />
            {Number(branchLogLength) > 0 && (
              <>
                <strong>Branch log ({branchLogLength})</strong>
                <HorizontalScrollContainer>
                  <Table className={classes.tableContainer}>
                    <TableHead>
                      <TableRow>
                        <TableHeaderCell />
                        <TableHeaderCell>Name</TableHeaderCell>
                        <TableHeaderCell>Snapshot ID</TableHeaderCell>
                        <TableHeaderCell>Data state at</TableHeaderCell>
                        <TableHeaderCell>Message</TableHeaderCell>
                      </TableRow>
                    </TableHead>
                    {snapshotList?.map((snapshot, id) => (
                      <TableBody key={id}>
                        {snapshot?.branch?.map((item, id) => (
                          <TableRow
                            key={id}
                            hover
                            className={classes.pointerCursor}
                            onClick={() =>
                              generateSnapshotPageId(snapshot.id) &&
                              history.push(
                                props.routes.snapshot(
                                  generateSnapshotPageId(snapshot.id) || '',
                                ),
                              )
                            }
                          >
                            <TableBodyCellMenu
                              actions={[
                                {
                                  name: 'Copy branch name',
                                  onClick: () => copyToClipboard(item),
                                },
                                {
                                  name: 'Copy snapshot ID',
                                  onClick: () =>
                                    copyToClipboard(snapshot.id || ''),
                                },
                              ]}
                            />
                            <TableBodyCell>{item}</TableBodyCell>
                            <TableBodyCell>{snapshot.id || '-'}</TableBodyCell>
                            <TableBodyCell>
                              {snapshot.dataStateAt || '-'}
                            </TableBodyCell>
                            <TableBodyCell>
                              {snapshot.message ?? '-'}
                            </TableBodyCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    ))}
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
            text={'Delete branch using CLI'}
          />
          <p className={classes.cliText}>
            You can delete this branch using CLI. To do this, run the command
            below:
          </p>
          <SyntaxHighlight content={`dblab branch delete ${props.branchId}`} />

          <SectionTitle
            className={classes.marginTop}
            tag="h2"
            level={2}
            text={'Get branches using CLI'}
          />
          <p className={classes.marginTop}>
            To list all branches using CLI, copy and paste it into your
            terminal.
          </p>
          <SyntaxHighlight content={getCliBranchListCommand()} />

          <SectionTitle
            className={classes.marginTop}
            tag="h2"
            level={2}
            text={'Get snapshots for this branch using CLI'}
          />
          <p className={classes.cliText}>
            You can get a list of snapshots for this branch using CLI. To do
            this, run the command below:
          </p>
          <SyntaxHighlight content={`dblab branch log ${props.branchId}`} />
        </div>
        <DeleteBranchModal
          isOpen={isOpenDestroyModal}
          onClose={() => setIsOpenDestroyModal(false)}
          deleteBranch={deleteBranch as DeleteBranch}
          branchName={props.branchId}
          instanceId={props.instanceId}
          afterSubmitClick={() => {
            stores.main.reload(props.branchId, props.instanceId)
            history.push(props.routes.branches())
          }}
        />
      </div>
    </>
  )
})
