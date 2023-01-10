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
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { icons } from '@postgres.ai/shared/styles/icons'
import { styles } from '@postgres.ai/shared/styles/styles'
import { DeleteBranchModal } from '@postgres.ai/shared/pages/Branches/components/Modals/DeleteBranchModal'
import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { generateSnapshotPageId } from '@postgres.ai/shared/pages/Instance/Snapshots/utils'
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
    textField: {
      ...styles.inputField,
      'max-width': 400,
      display: 'inline-block',
      '& .MuiOutlinedInput-input': {
        paddingRight: '32px!important',
      },
    },
    tableContainer: {
      position: 'relative',
      maxWidth: 400,
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
    deleteBranchError,
    getBranchError,
  } = stores.main

  const handleDestroyBranch = async () => {
    const isSuccess = await deleteBranch(props.branchId)
    if (isSuccess) history.push(props.routes.branch())
  }

  const hasBranchError = getBranchesError || getBranchError || snapshotListError

  const branchLogLength = snapshotList?.reduce((acc, snapshot) => {
    if (snapshot?.branch !== null) {
      return acc + snapshot.branch?.length
    } else {
      return acc
    }
  }, 0)

  const BranchHeader = () => {
    return (
      <>
        {props.elements.breadcrumbs}
        <SectionTitle
          className={classes.marginTop}
          tag="h1"
          level={1}
          text={`Branch ${props.branchId}`}
        />
      </>
    )
  }

  useEffect(() => {
    load(props.branchId)
  }, [])

  if (isBranchesLoading) return <PageSpinner />

  if (hasBranchError) {
    return (
      <>
        <BranchHeader />
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
      <BranchHeader />
      <div className={classes.container}>
        <div className={classes.actions}>
          <Button
            variant="contained"
            color="primary"
            onClick={() => setIsOpenDestroyModal(true)}
            disabled={isReloading}
            title={'Destroy this snapshot'}
            className={classes.actionButton}
          >
            Destroy branch
          </Button>
          <Button
            variant="outlined"
            color="secondary"
            onClick={() => reload(props.branchId)}
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
              <strong>Name</strong>
            </p>
            <p className={classes.text}>{branch?.name}</p>
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
            <p className={classes.text}>{branch?.dataStateAt || '-'}</p>
          </div>
          <div className={classes.summary}>
            <p>
              <strong>Summary</strong>&nbsp;
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
                      <TableHeaderCell>Data state at</TableHeaderCell>
                      <TableHeaderCell>Comment</TableHeaderCell>
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
                              `/instance/snapshots/${generateSnapshotPageId(
                                snapshot.id,
                              )}`,
                            )
                          }
                        >
                          <TableBodyCellMenu
                            actions={[
                              {
                                name: 'Copy branch name',
                                onClick: () => copyToClipboard(item),
                              },
                            ]}
                          />
                          <TableBodyCell>{item}</TableBodyCell>
                          <TableBodyCell>
                            {snapshot.dataStateAt || '-'}
                          </TableBodyCell>
                          <TableBodyCell>
                            {snapshot.comment ?? '-'}
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
        <DeleteBranchModal
          isOpen={isOpenDestroyModal}
          onClose={() => setIsOpenDestroyModal(false)}
          deleteBranchError={deleteBranchError}
          deleteBranch={handleDestroyBranch}
          branchName={props.branchId}
        />
      </div>
    </>
  )
})
