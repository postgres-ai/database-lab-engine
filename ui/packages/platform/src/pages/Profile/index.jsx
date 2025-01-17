/*--------------------------------------------------------------------------
* Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
* All Rights Reserved. Proprietary and confidential.
* Unauthorized copying of this file, via any medium is strictly prohibited
*--------------------------------------------------------------------------
*/

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Grid from '@material-ui/core/Grid';
import * as Yup from 'yup';
import { TextField } from '@postgres.ai/shared/components/TextField';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';

import Store from 'stores/store';
import Actions from 'actions/actions';
import { ErrorWrapper } from 'components/Error/ErrorWrapper';
import ConsolePageTitle from 'components/ConsolePageTitle';
import { Head, createTitle } from 'components/Head';
import CheckBoxOutlineBlankIcon from '@material-ui/icons/CheckBoxOutlineBlank';
import CheckBoxIcon from '@material-ui/icons/CheckBox';
import {Button, Checkbox, FormControlLabel, InputLabel} from "@material-ui/core";
import {Form, Formik} from "formik";

const PAGE_NAME = 'Profile';

const validationSchema = Yup.object({
  first_name: Yup.string().required('First name is required'),
  last_name: Yup.string().required('Last name is required'),
});

class Profile extends Component {
  componentDidMount() {
    const that = this;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const userProfile = this.data && this.data.userProfile ?
        this.data.userProfile : null;

      that.setState({ data: this.data });

      if (userProfile && !userProfile.isProcessing && userProfile.data.info) {
        that.setState({
          is_chats_email_notifications_enabled: userProfile.data.info.chats_email_notifications_enabled,
          first_name: userProfile.data.info.first_name,
          last_name: userProfile.data.info.last_name
        });
      }

      if (auth && auth.token && !userProfile.isProcessed && !userProfile.isProcessing &&
        !userProfile.error) {
        Actions.getUserProfile(auth.token);
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  handleSaveSettings = (values) => {
    const auth = this.state.data?.auth;
    if (auth) {
      Actions.updateUserProfile(auth.token, {
        is_chats_email_notifications_enabled: values.is_chats_email_notifications_enabled,
        first_name: values.first_name,
        last_name: values.last_name,
      });
    }
  };

  render() {
    const { classes } = this.props;
    const data = this.state && this.state.data ? this.state.data.userProfile : null;

    const initialValues = {
      first_name: data?.data?.info?.first_name || '',
      last_name: data?.data?.info?.last_name || '',
      is_chats_email_notifications_enabled: data?.data?.info?.chats_email_notifications_enabled || false,
    };


    const headRendered = (
      <Head title={createTitle([PAGE_NAME])} />
    );

    const pageTitle = (
      <ConsolePageTitle top title={PAGE_NAME}/>
    );

    if (this.state && this.state.data && this.state.data.userProfile.error) {
      return (
        <div>
          { headRendered }

          {pageTitle}

          <ErrorWrapper/>
        </div>
      );
    }

    if (!data || !data.data) {
      return (
        <div>
          { headRendered }

          {pageTitle}

          <PageSpinner className={classes.progress} />
        </div>
      );
    }

    return (
      <div className={classes.root}>
        { headRendered }

        {pageTitle}
        <Formik
          initialValues={initialValues}
          validationSchema={validationSchema}
          onSubmit={this.handleSaveSettings}
        >
          {({ values, handleChange, setFieldValue, errors, touched }) => (
            <Form>
              <Grid
                item
                xs={12}
                sm={6}
                md={6}
                lg={4}
                xl={3}
                className={classes.container}
              >
                <TextField
                  disabled
                  label='Email'
                  fullWidth
                  defaultValue={data.data.info.email}
                  className={classes.textField}
                />
                <TextField
                  label="First name"
                  fullWidth
                  name="first_name"
                  value={values.first_name}
                  onChange={handleChange}
                  className={classes.textField}
                  error={touched.first_name && !!errors.first_name}
                  helperText={touched.first_name && errors.first_name}
                />
                <TextField
                  label="Last name"
                  fullWidth
                  name="last_name"
                  value={values.last_name}
                  onChange={handleChange}
                  className={classes.textField}
                  error={touched.last_name && !!errors.last_name}
                  helperText={touched.last_name && errors.last_name}
                />
                <InputLabel className={classes.label} id="visibility-radio-buttons-group-label">
                  Notifications settings
                </InputLabel>
                <FormControlLabel
                  className={classes.formControlLabel}
                  control={
                    <Checkbox
                      icon={<CheckBoxOutlineBlankIcon fontSize="large" />}
                      checkedIcon={<CheckBoxIcon fontSize="large" />}
                      name="is_chats_email_notifications_enabled"
                      className={classes.formControlLabelCheckbox}
                      checked={values.is_chats_email_notifications_enabled}
                      onChange={(event) =>
                        setFieldValue('is_chats_email_notifications_enabled', event.target.checked)
                      }
                    />
                  }
                  label="Notify about new messages in the AI Assistant"
                />
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
                  disabled={data?.isProcessing}
                  id="userSaveButton"
                  type="submit"
                >
                  Save
                </Button>
              </Grid>
            </Form>
          )}
        </Formik>
      </div>
    );
  }
}

Profile.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default Profile
