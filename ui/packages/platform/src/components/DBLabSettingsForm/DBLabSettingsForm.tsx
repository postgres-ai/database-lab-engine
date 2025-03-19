import React, { useEffect, useMemo, useState } from 'react'
import { Link } from '@postgres.ai/shared/components/Link2'
import {
  Grid,
  Button,
  FormControl,
  FormControlLabel,
  makeStyles,
  Typography, TextField
} from '@material-ui/core'
import { useFormik } from "formik";
import * as Yup from 'yup';
import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import ConsolePageTitle from '../ConsolePageTitle'
import { DBLabSettingsFormProps } from './DBLabSettingsFormWrapper'
import { styles } from "@postgres.ai/shared/styles/styles";
import { PageSpinner } from "@postgres.ai/shared/components/PageSpinner";
import { WarningWrapper } from "../Warning/WarningWrapper";
import { messages } from "../../assets/messages";
import { ExternalIcon } from "@postgres.ai/shared/icons/External";
import Checkbox from '@mui/material/Checkbox/Checkbox'
import { hoursToPgInterval, pgIntervalToHours } from 'utils/utils';

type DBLabSettingsState = {
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
        dblab_low_disk_space_notifications_threshold_percent: number | null
        dblab_old_clones_notifications_threshold_hours: string | null
      }
    } | null
  } | null
}

interface NotificationsSettings {
  isLowDiskSpaceCheckboxActive: boolean;
  isOldClonesCheckboxActive: boolean;
  lowDiskSpaceThreshold: number | null | undefined;
  oldClonesThreshold: number | null | undefined;
}

export interface FormValues {
  notifications: NotificationsSettings;
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
    formContainer: {
      flexWrap: 'nowrap'
    },
    textField: {
      ...styles.inputField,
      marginBottom: 16,
      marginTop: 8
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
    }
  },
  { index: 1 },
)

const validationSchema = Yup.object({
  notifications: Yup.object({
    isLowDiskSpaceCheckboxActive: Yup.boolean().optional(),
    isOldClonesCheckboxActive: Yup.boolean().optional(),
    lowDiskSpaceThreshold: Yup.number()
      .nullable()
      .when('isLowDiskSpaceCheckboxActive', {
        is: true,
        then: (schema) => schema.required('Please enter a threshold value.').min(1, 'Must be at least 1'),
        otherwise: (schema) => schema.nullable(),
      }),
    oldClonesThreshold: Yup.number()
      .nullable()
      .when('isOldClonesCheckboxActive', {
        is: true,
        then: (schema) => schema.required('Please enter a threshold value.').min(1, 'Must be at least 1'),
        otherwise: (schema) => schema.nullable(),
      }),
  }),
});

const LOW_DISK_SPACE_THRESHOLD_DEFAULT = 20;
const OLD_CLONES_THRESHOLD_DEFAULT = 24;

const DBLabSettingsForm: React.FC<DBLabSettingsFormProps> = (props) => {
  const { orgPermissions, orgData, orgId, org, project } = props;
  const classes = useStyles();
  const [data, setData] = useState<DBLabSettingsState['data'] | null>(null);

  useEffect(() => {
    const unsubscribe = Store.listen(function () {
      const newStoreData = this.data;

      if (JSON.stringify(newStoreData) !== JSON.stringify(data)) {
        const auth = newStoreData?.auth || null;
        const orgProfile = newStoreData?.orgProfile || null;

        if (
          auth?.token &&
          orgProfile &&
          orgProfile.orgId !== orgId &&
          !orgProfile.isProcessing
        ) {
          Actions.getOrgs(auth.token, orgId);
        }


        setData(newStoreData);
      }
    });

    Actions.refresh();

    return () => {
      unsubscribe();
    };
  }, [orgId, data, props.match.params.projectId]);

  const isDBLabSettingsAvailable = useMemo(() => {
    const privileged_until = orgData?.priveleged_until;
    return !!(orgData && privileged_until && new Date(privileged_until) > new Date() && orgData?.consulting_type === 'enterprise');

  }, [orgData])



  const formik = useFormik<FormValues>({
    enableReinitialize: true,
    initialValues: {
      notifications: {
        isLowDiskSpaceCheckboxActive: Boolean(data?.orgProfile?.data?.dblab_low_disk_space_notifications_threshold_percent),
        isOldClonesCheckboxActive: Boolean(data?.orgProfile?.data?.dblab_old_clones_notifications_threshold_hours),
        lowDiskSpaceThreshold: data?.orgProfile?.data?.dblab_low_disk_space_notifications_threshold_percent || LOW_DISK_SPACE_THRESHOLD_DEFAULT,
        oldClonesThreshold: pgIntervalToHours(data?.orgProfile?.data?.dblab_old_clones_notifications_threshold_hours) || OLD_CLONES_THRESHOLD_DEFAULT,
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

      let params: { dblab_low_disk_space_notifications_threshold_percent: number | null, dblab_old_clones_notifications_threshold_hours: string | null } = {
        dblab_low_disk_space_notifications_threshold_percent: null,
        dblab_old_clones_notifications_threshold_hours: null
      }

      if (values.notifications.isLowDiskSpaceCheckboxActive) {
        params.dblab_low_disk_space_notifications_threshold_percent = values.notifications.lowDiskSpaceThreshold as number;
      }

      if (values.notifications.isOldClonesCheckboxActive) {
        params.dblab_old_clones_notifications_threshold_hours = hoursToPgInterval(values.notifications.oldClonesThreshold as number);
      }

      if (auth) {
        try {
          await Actions.updateDBLabSettings(auth.token, currentOrgId, params);
        } catch (error) {
          const errorMessage = `Error updating DBLab settings: ${error}`;
          Actions.showNotification(errorMessage, 'error');
          console.error('Error updating DBLab settings:', error);
        } finally {
          setSubmitting(false);
        }
      }
    }
  });


  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      org={org}
      project={project}
      breadcrumbs={[{ name: 'DBLab settings' }]}
    />
  );

  const pageTitle = <ConsolePageTitle title="DBLab settings" />;

  if (orgPermissions && !orgPermissions.settingsOrganizationUpdate) {
    return (
      <>
        {breadcrumbs}
        {pageTitle}
        <WarningWrapper>{messages.noPermissionPage}</WarningWrapper>
      </>
    );
  }

  if (!data || (data && data.orgProfile && data.orgProfile.isProcessing)) {
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
            <Grid container item lg={8} direction={'column'} className={classes.formContainer}>
              {!isDBLabSettingsAvailable && <Typography variant="body2" className={classes.unlockNote}>
                <Link external to="https://postgres.ai/contact/" target="_blank">
                  Become an Enterprise customer
                  <ExternalIcon className={classes.externalIcon}/>
                </Link>
                &nbsp;to unlock DBLab settings
              </Typography>}
              <Grid item xs={12} sm={6}>
                <FormControl fullWidth className={classes.selectField}>
                  <h2 className={classes.label}>E-mail notifications</h2>
                  <FormControlLabel
                    control={
                      <Checkbox
                        size="small"
                        checked={formik.values.notifications.isLowDiskSpaceCheckboxActive}
                        onChange={(e) =>
                          formik.setFieldValue(
                            'notifications.isLowDiskSpaceCheckboxActive',
                            e.target.checked
                          )
                        }
                      />
                    }
                    label="Notify organization administrators about low disk space" // TODO: @Nik, change text
                    disabled={!isDBLabSettingsAvailable}
                  />
                  <TextField
                    className={classes.textField}
                    id="lowDiskSpaceThresholdTextField"
                    label="Low disc space notification threshold (%)"
                    variant="outlined"
                    type="number"
                    disabled={!formik.values.notifications.isLowDiskSpaceCheckboxActive}
                    onChange={(e) => formik.setFieldValue('notifications.lowDiskSpaceThreshold', e.target.value)}
                    value={formik.values.notifications.lowDiskSpaceThreshold}
                    margin="normal"
                    fullWidth
                    inputProps={{
                      name: 'lowDiskSpaceThreshold',
                      id: 'lowDiskSpaceThresholdTextField',
                      min: 1,
                      max: 99,
                      inputMode: 'numeric',
                      pattern: '[0-9]*'
                    }}
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                    error={
                      formik.touched.notifications?.lowDiskSpaceThreshold &&
                      Boolean(formik.errors.notifications?.lowDiskSpaceThreshold)
                    }
                    helperText={
                      formik.touched.notifications?.lowDiskSpaceThreshold &&
                      formik.errors.notifications?.lowDiskSpaceThreshold
                    }
                  />
                  <FormControlLabel
                    control={
                      <Checkbox
                        size="small"
                        checked={formik.values.notifications.isOldClonesCheckboxActive}
                        onChange={(e) =>
                          formik.setFieldValue(
                            'notifications.isOldClonesCheckboxActive',
                            e.target.checked
                          )
                        }
                      />
                    }
                    label="Notify organization members about old clones" // TODO: @Nik, change text
                    disabled={!isDBLabSettingsAvailable}
                  />
                  <TextField
                    className={classes.textField}
                    id="oldClonesThresholdTextField"
                    label="Old clone threshold (hours)"
                    variant="outlined"
                    type="number"
                    disabled={!formik.values.notifications.isOldClonesCheckboxActive}
                    onChange={(e) => formik.setFieldValue('notifications.oldClonesThreshold', e.target.value)}
                    value={formik.values.notifications.oldClonesThreshold}
                    margin="normal"
                    fullWidth
                    inputProps={{
                      name: 'oldClonesThreshold',
                      id: 'oldClonesThresholdTextField',
                      min: 1,
                      inputMode: 'numeric',
                      pattern: '[0-9]*'
                    }}
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                    error={
                      formik.touched.notifications?.oldClonesThreshold &&
                      Boolean(formik.errors.notifications?.oldClonesThreshold)
                    }
                    helperText={
                      formik.touched.notifications?.oldClonesThreshold &&
                      formik.errors.notifications?.oldClonesThreshold
                    }
                  />
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
                disabled={data?.orgProfile?.isUpdating || !formik.dirty || !isDBLabSettingsAvailable}
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

export default DBLabSettingsForm
