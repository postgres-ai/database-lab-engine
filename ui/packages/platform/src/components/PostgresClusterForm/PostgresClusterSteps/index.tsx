import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Checkbox,
  FormControlLabel,
} from '@material-ui/core'
import { Box } from '@mui/material'
import { ReducerAction } from 'react'

import { ClassesType } from '@postgres.ai/platform/src/components/types'
import { icons } from '@postgres.ai/shared/styles/icons'
import { useCloudProviderProps } from 'hooks/useCloudProvider'

export const ClusterExtensionAccordion = ({
  step,
  state,
  classes,
  dispatch,
}: {
  step: number
  state: useCloudProviderProps['initialState']
  classes: ClassesType
  // @ts-ignore
  dispatch: ReducerAction<unknown, void>
}) => (
  <Accordion className={classes.sectionTitle}>
    <AccordionSummary
      aria-controls="extension-content"
      id="extension-header"
      expandIcon={icons.sortArrowDown}
    >
      {step}. Extensions
    </AccordionSummary>
    <AccordionDetails>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          width: '100%',
          fontWeight: 'normal',
        }}
      >
        <FormControlLabel
          className={classes.marginTop}
          control={
            <Checkbox
              name="pg_repack"
              checked={state.pg_repack}
              onChange={(e) =>
                dispatch({
                  type: 'change_pg_repack',
                  pg_repack: e.target.checked,
                })
              }
              classes={{
                root: classes.checkboxRoot,
              }}
            />
          }
          label={'pg_repack'}
        />
        <FormControlLabel
          control={
            <Checkbox
              name="pg_cron"
              checked={state.pg_cron}
              onChange={(e) =>
                dispatch({
                  type: 'change_pg_cron',
                  pg_cron: e.target.checked,
                })
              }
              classes={{
                root: classes.checkboxRoot,
              }}
            />
          }
          label={'pg_cron'}
        />
        <FormControlLabel
          control={
            <Checkbox
              name="pgaudit"
              checked={state.pgaudit}
              onChange={(e) =>
                dispatch({
                  type: 'change_pgaudit',
                  pgaudit: e.target.checked,
                })
              }
              classes={{
                root: classes.checkboxRoot,
              }}
            />
          }
          label={'pgaudit'}
        />
        {state.version !== 10 && (
          <FormControlLabel
            control={
              <Checkbox
                name="pgvector"
                checked={state.pgvector}
                onChange={(e) =>
                  dispatch({
                    type: 'change_pgvector',
                    pgvector: e.target.checked,
                  })
                }
                classes={{
                  root: classes.checkboxRoot,
                }}
              />
            }
            label={'pgvector'}
          />
        )}
        <FormControlLabel
          control={
            <Checkbox
              name="postgis"
              checked={state.postgis}
              onChange={(e) =>
                dispatch({
                  type: 'change_postgis',
                  postgis: e.target.checked,
                })
              }
              classes={{
                root: classes.checkboxRoot,
              }}
            />
          }
          label={'postgis'}
        />
        <FormControlLabel
          control={
            <Checkbox
              name="pgrouting"
              checked={state.pgrouting}
              onChange={(e) =>
                dispatch({
                  type: 'change_pgrouting',
                  pgrouting: e.target.checked,
                })
              }
              classes={{
                root: classes.checkboxRoot,
              }}
            />
          }
          label={'pgrouting'}
        />
        {state.version !== 10 && state.version !== 11 && (
          <FormControlLabel
            control={
              <Checkbox
                name="timescaledb"
                checked={state.timescaledb}
                onChange={(e) =>
                  dispatch({
                    type: 'change_timescaledb',
                    timescaledb: e.target.checked,
                  })
                }
                classes={{
                  root: classes.checkboxRoot,
                }}
              />
            }
            label={'timescaledb'}
          />
        )}
        {state.version !== 10 && (
          <FormControlLabel
            control={
              <Checkbox
                name="citus"
                checked={state.citus}
                onChange={(e) =>
                  dispatch({
                    type: 'change_citus',
                    citus: e.target.checked,
                  })
                }
                classes={{
                  root: classes.checkboxRoot,
                }}
              />
            }
            label={'citus'}
          />
        )}
        <FormControlLabel
          control={
            <Checkbox
              name="pg_partman"
              checked={state.pg_partman}
              onChange={(e) =>
                dispatch({
                  type: 'change_pg_partman',
                  pg_partman: e.target.checked,
                })
              }
              classes={{
                root: classes.checkboxRoot,
              }}
            />
          }
          label={'pg_partman'}
        />
        <FormControlLabel
          control={
            <Checkbox
              name="pg_stat_kcache"
              checked={state.pg_stat_kcache}
              onChange={(e) =>
                dispatch({
                  type: 'change_pg_stat_kcache',
                  pg_stat_kcache: e.target.checked,
                })
              }
              classes={{
                root: classes.checkboxRoot,
              }}
            />
          }
          label={'pg_stat_kcache'}
        />
        <FormControlLabel
          control={
            <Checkbox
              name="pg_wait_sampling"
              checked={state.pg_wait_sampling}
              onChange={(e) =>
                dispatch({
                  type: 'change_pg_wait_sampling',
                  pg_wait_sampling: e.target.checked,
                })
              }
              classes={{
                root: classes.checkboxRoot,
              }}
            />
          }
          label={'pg_wait_sampling'}
        />
      </Box>
    </AccordionDetails>
  </Accordion>
)
