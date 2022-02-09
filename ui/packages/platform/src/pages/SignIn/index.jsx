/*--------------------------------------------------------------------------
* Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
* All Rights Reserved. Proprietary and confidential.
* Unauthorized copying of this file, via any medium is strictly prohibited
*--------------------------------------------------------------------------
*/

import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Div100vh from 'react-div-100vh';

import Link from 'components/Link';
import settings from 'utils/settings';


const styles = theme => ({
  root: {
    fontFamily: 'dinpro-regular,sans-serif',
    backgroundColor: '#fbfbfb',
    overflow: 'auto',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 40,
    [theme.breakpoints.down('xs')]: {
      padding: '40px 24px'
    }
  },

  form: {
    flex: '0 1 400px',
    background: '#fff',
    border: '1px solid #c9d8db',
    margin: 'auto',
    borderRadius: '3px',
    padding: 40,
    [theme.breakpoints.down('xs')]: {
      padding: 24
    }
  },

  titleLink: {
    textDecoration: 'none',
    cursor: 'pointer'
  },

  title: {
    fontFamily: 'dinpro-light,sans-serif',
    color: '#1a1a1a',
    fontSize: '3rem',
    fontWeight: 300,
    marginBottom: 32,
    marginTop: 0,
    textAlign: 'center',
    [theme.breakpoints.down('xs')]: {
      fontSize: '2rem',
      marginBottom: 24
    }
  },

  button: {
    fontFamily: 'roboto,sans-serif',
    display: 'flex',
    height: 40,
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: 14,
    borderRadius: 5,
    backgroundColor: '#fff',
    color: 'rgba(0,0,0,.54)',
    border: '1px solid #cecece',
    fontWeight: 500,
    marginBottom: 16,
    textDecoration: 'none',
    padding: 12
  },

  buttonText: {
    marginLeft: 24,
    whiteSpace: 'nowrap',
    flex: '0 0 131px',

    [theme.breakpoints.down('xs')]: {
      marginLeft: 16
    }
  },

  terms: {
    fontFamily: 'inherit',
    paddingTop: 10,
    fontSize: 12,
    textAlign: 'center',
    lineHeight: '16px'
  }
});


const getAuthUrlFor = provider => `${settings.authUrl}/auth?provider=${provider}`;


const SignIn = (props) => {
  const { classes } = props;

  return (
    // See https://stackoverflow.com/questions/37112218/css3-100vh-not-constant-in-mobile-browser.
    <Div100vh className={ classes.root }>
      <div className={ classes.form }>
        <a className={ classes.titleLink } href={settings.rootUrl}>
          <h1 className={ classes.title }>Postgres.ai</h1>
        </a>

        <div>
          <a className={ classes.button } href={getAuthUrlFor('google')}>
            <img width='18' src='/images/oauth-google-logo.png' />
            <span className={ classes.buttonText }>Sign in with Google</span>
          </a>
          <a className={ classes.button } href={getAuthUrlFor('linkedin')}>
            <img width='18' src='/images/oauth-linkedin-logo.png' />
            <span className={ classes.buttonText }>Sign in with LinkedIn</span>
          </a>
          <a className={ classes.button } href={getAuthUrlFor('github')}>
            <img width='18' src='/images/oauth-github-logo.png' />
            <span className={ classes.buttonText }>Sign in with GitHub</span>
          </a>
          <a className={ classes.button } href={getAuthUrlFor('gitlab')}>
            <img width='18' src='/images/oauth-gitlab-logo.png' />
            <span className={ classes.buttonText }>Sign in with GitLab</span>
          </a>
        </div>

        <div className={ classes.terms }>
          <small>
            Signing in signifies that you have read and agree to our
            <br />
            <Link link={settings.rootUrl + '/tos'} target='_blank'>
              Terms&nbsp;of&nbsp;Service
            </Link>
            &nbsp;and&nbsp;
            <Link link={settings.rootUrl + '/privacy'} target='_blank'>
              Privacy&nbsp;Policy
            </Link>.
          </small>
        </div>
      </div>
    </Div100vh>
  );
};

SignIn.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(SignIn);
