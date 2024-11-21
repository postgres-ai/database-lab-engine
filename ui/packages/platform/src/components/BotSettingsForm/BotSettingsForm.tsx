/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useEffect, useMemo, useState } from 'react'
import { Link } from '@postgres.ai/shared/components/Link2'
import {
  Grid,
  Button,
  InputLabel,
  FormControl,
  FormControlLabel,
  makeStyles,
  Typography
} from '@material-ui/core'
import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import ConsolePageTitle from '../ConsolePageTitle'
import { BotSettingsFormProps } from './BotSettingsFormWrapper'
import { styles } from "@postgres.ai/shared/styles/styles";
import { PageSpinner } from "@postgres.ai/shared/components/PageSpinner";
import { WarningWrapper } from "../Warning/WarningWrapper";
import { messages } from "../../assets/messages";
import RadioGroup from '@mui/material/RadioGroup'
import Radio from '@mui/material/Radio'
import { ExternalIcon } from "@postgres.ai/shared/icons/External";
import { useFormik } from "formik";

type DbLabInstance = {
  id: number;
  plan: string | null;
}

type BotSettingState = {
  data: {
    auth: {
      token: string | null
    } | null
    orgProfile: {
      isUpdating: boolean
      error: boolean
      updateError: boolean
      errorMessage: string | undefined
      errorCode: number | undefined
      updateErrorMessage: string | null
      isProcessing: boolean
      orgId: number | null
      updateErrorFields: string[]
      data: {
        is_chat_public_by_default: boolean
      }
    } | null
    dbLabInstances: {
      data: Record<number, DbLabInstance>;
    }
  } | null
}

const useStyles = makeStyles(
  {
    container: {
      ...(styles.root as Object),
      display: 'flex',
      'flex-wrap': 'wrap',
      'min-height': 0,
      '&:not(:first-child)': {
        'margin-top': '20px',
      },
    },
    textField: {
      ...styles.inputField,
    },
    instructionsField: {
      ...styles.inputField,
    },
    selectField: {
      marginTop: 4,
      '& .MuiInputLabel-formControl': {
        transform: 'none',
        position: 'static'
      }
    },
    label: {
      color: '#000!important',
      fontWeight: 'bold',
    },
    radioGroup: {
      marginTop: 8
    },
    updateButtonContainer: {
      marginTop: 20,
      textAlign: 'left',
    },
    errorMessage: {
      color: 'red',
    },
    unlockNote: {
      marginTop: 8,
      '& ol': {
        paddingLeft: 24,
        marginTop: 6,
        marginBottom: 0
      }
    },
    formControlLabel: {
      '& .Mui-disabled > *, & .Mui-disabled': {
        color: 'rgba(0, 0, 0, 0.6)'
      }
    },
    externalIcon: {
      width: 14,
      height: 14,
      marginLeft: 4,
      transform: 'translateY(2px)',
    }
  },
  { index: 1 },
)

const BotSettingsForm: React.FC<BotSettingsFormProps> = (props) => {
  const { orgPermissions, orgData, orgId, org, project } = props;

  const classes = useStyles()

  const [threadVisibility, setThreadVisibility] = useState('private')
  const [data, setData] = useState<BotSettingState['data'] | null>(null)

  const isSubscriber = useMemo(() => {
    const hasEEPlan = orgData?.data?.plan === 'EE';
    const hasSEPlan = data?.dbLabInstances?.data
      ? Object.values(data.dbLabInstances.data).some((item) => item.plan === "SE")
      : false;

    return hasEEPlan || hasSEPlan;
  }, [data?.dbLabInstances?.data, orgData?.data?.plan]);


  useEffect(() => {
    const unsubscribe = Store.listen(function () {
      const newStoreData = this.data;

      if (JSON.stringify(newStoreData) !== JSON.stringify(data)) {
        const auth = newStoreData?.auth || null;
        const orgProfile = newStoreData?.orgProfile || null;
        const dbLabInstances = newStoreData?.dbLabInstances || null;
        const projectId = props.match?.params?.projectId || null;

        if (
          auth?.token &&
          orgProfile &&
          orgProfile.orgId !== orgId &&
          !orgProfile.isProcessing
        ) {
          Actions.getOrgs(auth.token, orgId);
        }

        if (
          auth?.token &&
          !dbLabInstances?.isProcessing &&
          !dbLabInstances?.error
        ) {
          Actions.getDbLabInstances(auth.token, orgId, projectId);
        }

        setData(newStoreData);
      }
    });

    Actions.refresh();

    return () => {
      unsubscribe();
    };
  }, [orgId, data, props.match.params.projectId]);

  const formik = useFormik({
    enableReinitialize: true,
    initialValues: {
      threadVisibility:
        data?.orgProfile?.data?.is_chat_public_by_default ? 'public' : 'private',
    },
    onSubmit: () => {
      const currentOrgId = orgId || null;
      const auth = data?.auth || null;

      if (auth) {
        let params: { is_chat_public_by_default?: boolean } = {
          is_chat_public_by_default:
            formik.values.threadVisibility === 'public',
        };

        Actions.updateAiBotSettings(auth.token, currentOrgId, params);
      }
    },
  });

  const handleChangeThreadVisibility = (
    event: React.ChangeEvent<{ value: string }>
  ) => {
    formik.handleChange(event);
  };

  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      org={org}
      project={project}
      breadcrumbs={[{ name: 'AI Assistant' }]}
    />
  )

  const pageTitle = (
    <ConsolePageTitle title="AI Assistant settings" />
  )

  if (orgPermissions && !orgPermissions.settingsOrganizationUpdate) {
    return (
      <>
        {breadcrumbs}

        {pageTitle}

        <WarningWrapper>{messages.noPermissionPage}</WarningWrapper>
      </>
    )
  }

  if (!data || (data && data.orgProfile && data.orgProfile.isProcessing)) {
    return (
      <div>
        {breadcrumbs}

        {pageTitle}

        <PageSpinner/>
      </div>
    )
  }

  return (
    <>
      {breadcrumbs}

      {pageTitle}
      <form onSubmit={formik.handleSubmit}>
        <Grid container spacing={3}>
          <Grid item xs={12} sm={12} lg={12} className={classes.container}>
            <Grid container item xs={12} sm={12} lg={8} direction={'column'}>
              <Grid item xs={12} sm={6}>
                <FormControl fullWidth className={classes.selectField}>
                  <InputLabel className={classes.label} id="visibility-radio-buttons-group-label">
                    AI chats default visibility
                  </InputLabel>
                  <RadioGroup
                    aria-labelledby="visibility-radio-buttons-group-label"
                    defaultValue="public"
                    name="threadVisibility"
                    value={formik.values.threadVisibility}
                    onChange={handleChangeThreadVisibility}
                    className={classes.radioGroup}
                  >
                    <FormControlLabel
                      value="public"
                      className={classes.formControlLabel}
                      control={<Radio size="small"/>}
                      label={<><b>Public:</b> anyone can view chats, but only team members can respond</>}
                    />
                    <FormControlLabel
                      disabled={!isSubscriber}
                      className={classes.formControlLabel}
                      value="private"
                      control={<Radio size="small"/>}
                      label={<><b>Private:</b> chats are visible only to members of your organization</>}
                    />
                    {!isSubscriber && <Typography variant="body2" className={classes.unlockNote}>
                      Unlock private conversations by either:
                      <ol>
                        <li>
                          <Link to={`/${org}/instances`} target="_blank">
                            Installing a DBLab SE instance
                            <ExternalIcon className={classes.externalIcon}/>
                          </Link>
                        </li>
                        <li>
                          <Link external to="https://postgres.ai/consulting" target="_blank">
                            Becoming a Postgres.AI consulting customer
                            <ExternalIcon className={classes.externalIcon}/>
                          </Link>
                        </li>
                      </ol>
                    </Typography>}
                  </RadioGroup>
                </FormControl>
              </Grid>
            </Grid>
            <Grid
              item
              xs={12}
              sm={12}
              lg={8}
              className={classes.updateButtonContainer}
            >
              <Button
                variant="contained"
                color="primary"
                disabled={data?.orgProfile?.isUpdating || !formik.dirty || !formik.isValid}
                id="orgSaveButton"
                type="submit"
              >
                Save
              </Button>
            </Grid>
          </Grid>
        </Grid>
      </form>
    </>
  )
}

export default BotSettingsForm
