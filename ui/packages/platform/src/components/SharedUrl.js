/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Button from '@material-ui/core/Button';

import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { icons } from '@postgres.ai/shared/styles/icons';

import JoeSessionCommand from 'pages/JoeSessionCommand';

import Actions from '../actions/actions';
import Error from './Error';
import Store from '../stores/store';
import settings from '../utils/settings';


const SIGNUP_BANNER_PARAM = 'signUpBannerClosed';

const styles = theme => ({
  container: {
    display: 'flex',
    flexWrap: 'wrap'
  },
  textField: {
    marginLeft: theme.spacing(1),
    marginRight: theme.spacing(1),
    width: '80%'
  },
  dense: {
    marginTop: 16
  },
  menu: {
    width: 200
  },
  updateButtonContainer: {
    marginTop: 20,
    textAlign: 'right'
  },
  errorMessage: {
    color: 'red'
  },
  orgsHeader: {
    position: 'relative'
  },
  newOrgBtn: {
    position: 'absolute',
    top: 0,
    right: 10
  },
  banner: {
    'height': 50,
    'position': 'absolute',
    'left': 0,
    'bottom': 0,
    'right': 0,
    'backgroundColor': 'rgba(1, 58, 68, 0.8)',
    'color': 'white',
    'zIndex': 100,
    'fontSize': 18,
    'lineHeight': '50px',
    'paddingLeft': 20,
    '& a': {
      color: 'white',
      fontWeight: '600'
    },
    '& svg': {
      position: 'absolute',
      right: 18,
      top: 18,
      cursor: 'pointer'
    }
  },
  signUpButton: {
    'backgroundColor': 'white',
    'fontWeight': 600,
    'marginLeft': 10,
    '&:hover': {
      backgroundColor: '#ecf6f7'
    }
  }
});

class SharedUrl extends Component {
  state = {
    signUpBannerClosed: localStorage.getItem(SIGNUP_BANNER_PARAM) === '1'
  };

  componentDidMount() {
    const that = this;
    const uuid = this.props.match.params.url_uuid;

    this.unsubscribe = Store.listen(function () {
      const sharedUrlData = this.data && this.data.sharedUrlData ?
        this.data.sharedUrlData : null;

      that.setState({ data: this.data });

      if (!sharedUrlData.isProcessing && !sharedUrlData.error &&
        !sharedUrlData.isProcessed) {
        Actions.getSharedUrlData(uuid);
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  closeBanner = () => {
    localStorage.setItem(SIGNUP_BANNER_PARAM, 1);
    this.setState({ signUpBannerClosed: true });
  };

  signUp = () => {
    window.open(
      settings.signinUrl,
      '_blank'
    );
  };

  render() {
    const { classes } = this.props;
    const data = this.state && this.state.data && this.state.data.sharedUrlData ?
      this.state.data.sharedUrlData : null;
    const env = this.state && this.state.data ? this.state.data.userProfile : null;
    const showBanner = !this.state.signUpBannerClosed;

    if (!data || (data && (data.isProcessing || !data.isProcessed))) {
      return (
        <>
          <PageSpinner />
        </>
      );
    }

    if (data && data.isProcessed && data.error) {
      return (
        <>
          <Error
            code={404}
            message={'Not found.'}/>
        </>
      );
    }

    let page = null;
    if (data.data.url.object_type === 'command') {
      page = (
        <JoeSessionCommand
          {...this.props}
          commandId={data.data.url.object_id}
          sessionId={data.data.url_data.joe_session_id}
        />
      );
    }

    let banner = null;
    if (!env || (env && !env.data)) {
      banner = (
        <div className={classes.banner}>
          Boost your development process with&nbsp;
          <a target='_blank' href='https://postgres.ai'>Postgres.ai Platform</a>&nbsp;
          <Button
            onClick={ () => this.signUp() }
            variant='outlined'
            color='secondary'
            className={classes.signUpButton}
          >
            Sign up
          </Button>

          <span
            className={classes.bannerCloseButton}
            onClick={this.closeBanner}
          >
            {icons.bannerCloseIcon}
          </span>
        </div>
      );
    }

    return (
      <>
        <style>
          {`
            .intercom-lightweight-app-launcher,
            iframe.intercom-launcher-frame {
              bottom: 30px!important;
              right: 30px!important;
            }
          `}
        </style>
        {page}
        {showBanner && banner}
      </>
    );
  }
}

SharedUrl.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(SharedUrl);
