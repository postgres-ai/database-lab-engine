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
  FormControl,
  FormControlLabel,
  makeStyles,
  Typography
} from '@material-ui/core'
import * as Yup from 'yup';
import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import ConsolePageTitle from '../ConsolePageTitle'
import { AuditSettingsFormProps } from './AuditSettingsFormWrapper'
import { styles } from "@postgres.ai/shared/styles/styles";
import { PageSpinner } from "@postgres.ai/shared/components/PageSpinner";
import { WarningWrapper } from "../Warning/WarningWrapper";
import { messages } from "../../assets/messages";
import { ExternalIcon } from "@postgres.ai/shared/icons/External";
import { useFormik } from "formik";
import Checkbox from '@mui/material/Checkbox/Checkbox'
import { SIEMIntegrationForm } from "../SIEMIntegrationForm/SIEMIntegrationForm";

type AuditSettingState = {
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
        siem_integration_enabled: SiemSettings["enableSiemIntegration"]
        siem_integration_url: SiemSettings["urlSchema"]
        siem_integration_request_headers: SiemSettings["headers"]
        audit_events_to_log: string[]
      }
    } | null
    auditEvents: {
      isProcessing: boolean
      data: {
        id: number
        event_name: string
        label: string
      }[] | null
    } | null
  } | null
}

interface SiemSettings {
  enableSiemIntegration: boolean;
  urlSchema?: string;
  headers: { key: string; value: string }[];
  auditEvents: EventsToLog[];
}

interface EventsToLog {
  id: number;
  event_name: string;
  label: string;
}

export interface FormValues {
  siemSettings: SiemSettings;
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

    },
    label: {
      color: '#000!important',
      fontWeight: 'bold',
    },
    updateButtonContainer: {
      marginTop: 20,
      textAlign: 'left',
    },
    unlockNote: {
      marginTop: 8,
      '& ol': {
        paddingLeft: 24,
        marginTop: 6,
        marginBottom: 0
      }
    },
    externalIcon: {
      width: 14,
      height: 14,
      marginLeft: 4,
      transform: 'translateY(2px)',
    },
    testConnectionButton: {
      marginRight: 16
    },
    eventRow: {
      display: 'flex',
      alignItems: 'center',
      marginBottom: '10px',
    },
  },
  { index: 1 },
)

const validationSchema = Yup.object({
  siemSettings: Yup.object({
    urlSchema: Yup.string()
      .url('Invalid URL format') // Validates that the input is a valid URL
      .required('URL is required'), // Field is mandatory
    headers: Yup.array().of(
      Yup.object({
        key: Yup.string().optional(),
        value: Yup.string().optional(),
      })
    ),
    auditEvents: Yup.array()
  }),
});

const AuditSettingsForm: React.FC<AuditSettingsFormProps> = (props) => {
  const { orgPermissions, orgData, orgId, org, project } = props;
  const classes = useStyles();
  const [data, setData] = useState<AuditSettingState['data'] | null>(null);

  useEffect(() => {
    const unsubscribe = Store.listen(function () {
      const newStoreData = this.data;

      if (JSON.stringify(newStoreData) !== JSON.stringify(data)) {
        const auth = newStoreData?.auth || null;
        const orgProfile = newStoreData?.orgProfile || null;
        const auditEvents = newStoreData?.auditEvents || null;

        if (
          auth?.token &&
          orgProfile &&
          orgProfile.orgId !== orgId &&
          !orgProfile.isProcessing
        ) {
          Actions.getOrgs(auth.token, orgId);
        }

        if (auth?.token && auditEvents && !auditEvents.isProcessing) {
          Actions.getAuditEvents(auth.token);
        }

        setData(newStoreData);
      }
    });

    Actions.refresh();

    return () => {
      unsubscribe();
    };
  }, [orgId, data, props.match.params.projectId]);

  const isAuditLogsSettingsAvailable = useMemo(() => {
    const privileged_until = orgData?.priveleged_until;
    return !!(orgData && privileged_until && new Date(privileged_until) > new Date() && orgData?.data?.plan === 'EE');

  }, [orgData])

  const formik = useFormik<FormValues>({
    enableReinitialize: true,
    initialValues: {
      siemSettings: {
        enableSiemIntegration: Boolean(data?.orgProfile?.data?.siem_integration_enabled),
        urlSchema: data?.orgProfile?.data?.siem_integration_url || '',
        headers: data?.orgProfile?.data?.siem_integration_request_headers
          ? Object.entries(data.orgProfile.data.siem_integration_request_headers).map(([key, value]) => ({
              key: key || '',
              value: value || '',
            })) as unknown as SiemSettings['headers']
          : [{ key: '', value: '' }],
        auditEvents: data?.auditEvents?.data
          ? data?.auditEvents?.data
              ?.filter((event) =>
                data?.orgProfile?.data?.audit_events_to_log?.includes(event.event_name)
              )
              ?.map((event) => ({
                id: event.id,
                event_name: event.event_name,
                label: event.label,
              }))
          : [],
      },
    },
    validationSchema,
    onSubmit: async (values, { setSubmitting }) => {
      const errors = await formik.validateForm();

      if (Object.keys(errors).length > 0) {
        console.error('Validation errors:', errors);
        setSubmitting(false);
        return; // Stop submission if there are errors
      }

      const currentOrgId = orgId || null;
      const auth = data?.auth || null;

      if (auth) {
        const params = formik.values.siemSettings;
        try {
          await Actions.updateAuditSettings(auth.token, currentOrgId, params);
        } catch (error) {
          const errorMessage = `Error updating audit settings: ${error}`;
          Actions.showNotification(errorMessage, 'error');
          console.error('Error updating audit settings:', error);
        } finally {
          setSubmitting(false);
        }
      }
    }
  });

  const isDisabled = useMemo(() =>
    !isAuditLogsSettingsAvailable || !formik.values.siemSettings.enableSiemIntegration,
    [isAuditLogsSettingsAvailable, formik.values.siemSettings.enableSiemIntegration]
  );

  const testConnection = async () => {
    try {
      const auth = data?.auth || null;

      if (auth) {
        const params = {...formik.values.siemSettings};
        if (formik.values.siemSettings.urlSchema) {
          Actions.testSiemServiceConnection(auth.token, params);
        }
      }
    } catch (error) {
      console.error('Connection failed:', error);
    }
  };

  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      org={org}
      project={project}
      breadcrumbs={[{ name: 'Audit' }]}
    />
  );

  const pageTitle = <ConsolePageTitle title="Audit settings" />;

  if (orgPermissions && !orgPermissions.settingsOrganizationUpdate) {
    return (
      <>
        {breadcrumbs}
        {pageTitle}
        <WarningWrapper>{messages.noPermissionPage}</WarningWrapper>
      </>
    );
  }

  if (!data || (data && data.orgProfile && data.orgProfile.isProcessing) || (data && data.auditEvents && data.auditEvents.isProcessing)) {
    return (
      <div>
        {breadcrumbs}
        {pageTitle}
        <PageSpinner />
      </div>
    );
  }

  return (
    <>
      {breadcrumbs}
      {pageTitle}
      <form onSubmit={formik.handleSubmit}>
        <Grid container spacing={3}>
          <Grid item xs={12} sm={12} lg={12} className={classes.container}>
            <Grid container item xs={12} sm={12} lg={8} direction={'column'}>
              {!isAuditLogsSettingsAvailable && <Typography variant="body2" className={classes.unlockNote}>
                <Link external to="https://postgres.ai/contact/" target="_blank">
                  Become an Enterprise customer
                  <ExternalIcon className={classes.externalIcon}/>
                </Link>
                &nbsp;to unlock audit settings
              </Typography>}
              <Typography variant="body2" className={classes.unlockNote}>
                <Link external to="https://postgres.ai/docs/how-to-guides/platform/audit-logs" target="_blank">
                  SIEM audit logs integration documentation
                  <ExternalIcon className={classes.externalIcon}/>
                </Link>
              </Typography>
              <Grid item xs={12} sm={6}>
                <FormControl fullWidth className={classes.selectField}>
                  <h2 className={classes.label}>SIEM integration</h2>
                  <FormControlLabel
                    control={
                      <Checkbox
                        size="small"
                        checked={formik.values.siemSettings.enableSiemIntegration}
                        onChange={(e) =>
                          formik.setFieldValue(
                            'siemSettings.enableSiemIntegration',
                            e.target.checked
                          )
                        }
                      />
                    }
                    label="Send audit events to SIEM system"
                    disabled={!isAuditLogsSettingsAvailable}
                  />
                  <h2 className={classes.label}>SIEM connection settings</h2>
                  <SIEMIntegrationForm formik={formik} disabled={isDisabled} />
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
                disabled={data?.orgProfile?.isUpdating || !formik.isValid || isDisabled}
                id="testConnectionButton"
                className={classes.testConnectionButton}
                onClick={testConnection}
              >
                Test connection
              </Button>
            </Grid>
            <Grid container item xs={12} sm={12} lg={8} direction={'column'}>
              <Grid item xs={12} sm={6}>
                <FormControl fullWidth className={classes.selectField}>
                  <h2 className={classes.label}>Select audit events to export</h2>
                  {data?.auditEvents?.data &&
                    data?.auditEvents?.data?.map((event) => {
                      const isChecked = formik.values.siemSettings.auditEvents.some(
                        (e) => e.event_name === event.event_name
                      );

                      return (
                        <div key={event.id} className={classes.eventRow}>
                          <FormControlLabel
                            control={
                              <Checkbox
                                size="small"
                                checked={isChecked}
                                onChange={(e) => {
                                  const updatedAuditEvents = e.target.checked
                                    ? [...formik.values.siemSettings.auditEvents, { ...event }]
                                    : formik.values.siemSettings.auditEvents.filter(
                                        (auditEvent) => auditEvent.event_name !== event.event_name
                                      );

                                  formik.setFieldValue('siemSettings.auditEvents', updatedAuditEvents);
                                }}
                              />
                            }
                            label={event.label}
                            disabled={isDisabled}
                          />
                        </div>
                      );
                    })}
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
                disabled={data?.orgProfile?.isUpdating || !formik.dirty || !isAuditLogsSettingsAvailable}
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
  );
};

export default AuditSettingsForm
