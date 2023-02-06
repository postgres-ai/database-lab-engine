/*--------------------------------------------------------------------------
* Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
* All Rights Reserved. Proprietary and confidential.
* Unauthorized copying of this file, via any medium is strictly prohibited
*--------------------------------------------------------------------------
*/

import PropTypes from 'prop-types';
import Div100vh from 'react-div-100vh';
import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink';

import settings from 'utils/settings';

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
            <img width='18' src='/images/oauth-google-logo.png' alt="Google sign in" />
            <span className={ classes.buttonText }>Sign in with Google</span>
          </a>
          <a className={ classes.button } href={getAuthUrlFor('linkedin')}>
            <img width='18' src='/images/oauth-linkedin-logo.png' alt="LinkedIn sign in" />
            <span className={ classes.buttonText }>Sign in with LinkedIn</span>
          </a>
          <a className={ classes.button } href={getAuthUrlFor('github')}>
            <img width='18' src='/images/oauth-github-logo.png' alt="Github sign in" />
            <span className={ classes.buttonText }>Sign in with GitHub</span>
          </a>
          <a className={ classes.button } href={getAuthUrlFor('gitlab')}>
            <img width='18' src='/images/oauth-gitlab-logo.png' alt="Gitlab sign in" />
            <span className={ classes.buttonText }>Sign in with GitLab</span>
          </a>
        </div>

        <div className={ classes.terms }>
          <small>
            Signing in signifies that you have read and agree to our
            <br />
            <GatewayLink href={settings.rootUrl + '/tos'} target='_blank'>
              Terms&nbsp;of&nbsp;Service
            </GatewayLink>
            &nbsp;and&nbsp;
            <GatewayLink href={settings.rootUrl + '/privacy'} target='_blank'>
              Privacy&nbsp;Policy
            </GatewayLink>.
          </small>
        </div>
      </div>
    </Div100vh>
  );
};

SignIn.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default SignIn;
