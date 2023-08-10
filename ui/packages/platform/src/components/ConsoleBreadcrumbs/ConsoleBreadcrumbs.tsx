/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import { NavLink } from 'react-router-dom'
import { Typography, Paper, Breadcrumbs } from '@material-ui/core'
import clsx from 'clsx'

import { Head, createTitle as createTitleBase } from 'components/Head'
import { ConsoleBreadcrumbsProps } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import Urls from '../../utils/urls'

interface ConsoleBreadcrumbsState {
  data: {
    userProfile: {
      data: {
        orgs: {
          [org: string]: {
            name: string
            projects: {
              [project: string]: {
                name: string
              }
            }
          }
        }
      }
    }
  }
}

interface ConsoleBreadcrumbsWithStylesProps extends ConsoleBreadcrumbsProps {
  classes: ClassesType
}

const createTitle = (parts: string[]) => {
  const filteredParts = parts.filter((part) => part !== 'Organizations')
  return createTitleBase(filteredParts)
}

class ConsoleBreadcrumbs extends Component<
  ConsoleBreadcrumbsWithStylesProps,
  ConsoleBreadcrumbsState
> {
  unsubscribe: Function
  componentDidMount() {
    const that = this

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      that.setState({ data: this.data })
    })
    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  render() {
    const { classes, hasDivider = false, breadcrumbs } = this.props
    const org = this.props.org ? this.props.org : null
    const project = this.props.project ? this.props.project : null
    const orgs =
      this.state &&
      this.state.data &&
      this.state.data.userProfile.data &&
      this.state.data.userProfile.data.orgs
        ? this.state.data.userProfile.data.orgs
        : null
    const paths = []
    let lastUrl = ''

    if (!breadcrumbs.length || Urls.isSharedUrl()) {
      return null
    }

    if (org && orgs && orgs[org]) {
      if (orgs[org].name) {
        paths.push({ name: 'Organizations', url: '/' })
        paths.push({ name: orgs[org].name, url: '/' + org })
        lastUrl = '/' + org
      }

      if (project && orgs[org].projects && orgs[org].projects[project]) {
        paths.push({ name: orgs[org].projects[project].name, url: null })
        lastUrl = '/' + org + '/' + project
      }
    }

    for (let i = 0; i < breadcrumbs.length; i++) {
      if (breadcrumbs[i].url && breadcrumbs[i].url?.indexOf('/') === -1) {
        breadcrumbs[i].url = lastUrl + '/' + breadcrumbs[i].url
        lastUrl = breadcrumbs[i].url as string
      }
      breadcrumbs[i].isLast = i === breadcrumbs.length - 1
      paths.push(breadcrumbs[i])
    }

    return (
      <>
        <Head title={createTitle(paths.map((path) => path.name))} />
        <Paper
          elevation={0}
          className={clsx(
            classes?.breadcrumbPaper,
            hasDivider && classes?.breadcrumbPaperWithDivider,
          )}
        >
          <Breadcrumbs aria-label="breadcrumb">
            {paths.map(
              (
                b: { name: string; url?: string | null; isLast?: boolean },
                index,
              ) => {
                return (
                  <span key={index}>
                    {b.url ? (
                      <NavLink
                        color="inherit"
                        to={b.url}
                        className={classes?.breadcrumbsLink}
                      >
                        {b.name}
                      </NavLink>
                    ) : (
                      <Typography
                        color="textPrimary"
                        className={
                          b.isLast
                            ? classes?.breadcrumbsActiveItem
                            : classes?.breadcrumbsItem
                        }
                      >
                        {b.name}
                      </Typography>
                    )}
                  </span>
                )
              },
            )}
          </Breadcrumbs>
        </Paper>
      </>
    )
  }
}

export default ConsoleBreadcrumbs
