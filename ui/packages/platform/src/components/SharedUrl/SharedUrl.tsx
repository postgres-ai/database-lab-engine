/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import { Button } from '@material-ui/core'

import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { icons } from '@postgres.ai/shared/styles/icons'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import { JoeSessionCommandWrapper } from 'pages/JoeSessionCommand/JoeSessionCommandWrapper'
import Actions from '../../actions/actions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import Store from '../../stores/store'
import settings from '../../utils/settings'
import { SharedUrlProps } from 'components/SharedUrl/SharedUrlWrapper'

interface SharedUrlWithStylesProps extends SharedUrlProps {
  classes: ClassesType
}

interface SharedUrlState {
  signUpBannerClosed: boolean
  data: {
    sharedUrlData: {
      isProcessing: boolean
      isProcessed: boolean
      error: boolean
      data: {
        url: {
          object_type: string | null
          object_id: number | null
        }
        url_data: {
          joe_session_id: number | null
        }
      }
    } | null
    userProfile: { data: Object | null } | null
  } | null
}

const SIGN_UP_BANNER_PARAM = 'signUpBannerClosed'

class SharedUrl extends Component<SharedUrlWithStylesProps, SharedUrlState> {
  state = {
    signUpBannerClosed: localStorage.getItem(SIGN_UP_BANNER_PARAM) === '1',
    data: {
      sharedUrlData: {
        isProcessing: false,
        isProcessed: false,
        error: false,
        data: {
          url: {
            object_type: null,
            object_id: null,
          },
          url_data: { joe_session_id: null },
        },
      },
      userProfile: {
        data: null,
      },
    },
  }

  unsubscribe: Function
  componentDidMount() {
    const that = this
    const uuid = this.props.match.params.url_uuid

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      const sharedUrlData =
        this.data && this.data.sharedUrlData ? this.data.sharedUrlData : null

      that.setState({ data: this.data })

      if (
        !sharedUrlData?.isProcessing &&
        !sharedUrlData?.error &&
        !sharedUrlData?.isProcessed
      ) {
        Actions.getSharedUrlData(uuid)
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  closeBanner = () => {
    localStorage.setItem(SIGN_UP_BANNER_PARAM, '1')
    this.setState({ signUpBannerClosed: true })
  }

  signUp = () => {
    window.open(settings.signinUrl, '_blank')
  }

  render() {
    const { classes } = this.props
    const data =
      this.state && this.state.data && this.state.data.sharedUrlData
        ? this.state.data.sharedUrlData
        : null
    const env =
      this.state && this.state.data ? this.state.data.userProfile : null
    const showBanner = !this.state.signUpBannerClosed

    if (!data || (data && (data.isProcessing || !data.isProcessed))) {
      return (
        <>
          <PageSpinner />
        </>
      )
    }

    if (data && data.isProcessed && data.error) {
      return (
        <>
          <ErrorWrapper code={404} message={'Not found.'} />
        </>
      )
    }

    let page = null
    if (data?.data?.url && data.data.url?.object_type === 'command') {
      const customProps = {
        commandId: data.data.url.object_id,
        sessionId: data.data.url_data.joe_session_id,
      }
      page = <JoeSessionCommandWrapper {...this.props} {...customProps} />
    }

    let banner = null
    if (!env || (env && !env.data)) {
      banner = (
        <div className={classes.banner}>
          Boost your development process with&nbsp;
          <a target="_blank" href="https://postgres.ai" rel="noreferrer">
            Postgres.ai Platform
          </a>
          &nbsp;
          <Button
            onClick={() => this.signUp()}
            variant="outlined"
            color="secondary"
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
      )
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
    )
  }
}

export default SharedUrl
