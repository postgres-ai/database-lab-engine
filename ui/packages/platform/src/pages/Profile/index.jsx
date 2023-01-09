/*--------------------------------------------------------------------------
* Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
* All Rights Reserved. Proprietary and confidential.
* Unauthorized copying of this file, via any medium is strictly prohibited
*--------------------------------------------------------------------------
*/

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Grid from '@material-ui/core/Grid';

import { TextField } from '@postgres.ai/shared/components/TextField';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';

import Store from 'stores/store';
import Actions from 'actions/actions';
import { ErrorWrapper } from 'components/Error/ErrorWrapper';
import ConsolePageTitle from 'components/ConsolePageTitle';
import { Head, createTitle } from 'components/Head';

const PAGE_NAME = 'Profile';

class Profile extends Component {
  componentDidMount() {
    const that = this;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const userProfile = this.data && this.data.userProfile ?
        this.data.userProfile : null;

      that.setState({ data: this.data });

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

  render() {
    const { classes } = this.props;
    const data = this.state && this.state.data ? this.state.data.userProfile : null;

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

        <Grid
          item
          xs={12}
          sm={6}
          md={4}
          lg={3}
          xl={2}
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
            disabled
            label='First name'
            fullWidth
            defaultValue={data.data.info.first_name}
            className={classes.textField}
          />
          <TextField
            disabled
            label='Last name'
            fullWidth
            defaultValue={data.data.info.last_name}
            className={classes.textField}
          />
        </Grid>
      </div>
    );
  }
}

Profile.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default Profile
