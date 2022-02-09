/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles, Dialog, IconButton } from '@material-ui/core'
import { Close as CloseIcon } from '@material-ui/icons'
import clsx from 'clsx'

import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { colors } from '@postgres.ai/shared/styles/colors'

type Props = {
  isOpen: boolean
  onClose: () => void
  children: React.ReactNode
  title: React.ReactNode
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  titleRightContent?: React.ReactNode
  headerContent?: React.ReactNode
  classes?: {
    content?: string
  }
}

const useStyles = makeStyles((theme) => ({
  root: {
    padding: '24px',
    width: '100%',

    [theme.breakpoints.down('xs')]: {
      padding: '20px',
      margin: '20px',
      maxHeight: 'calc(100% - 40px)',
    },
  },
  closeButton: {
    position: 'absolute',
    right: '12px',
    top: '12px',
    color: colors.pgaiDarkGray,
  },
  titleContent: {
    paddingRight: '36px'
  },
  content: {
    marginTop: '16px',
  },
}))

export const Modal = (props: Props) => {
  const {
    isOpen,
    onClose,
    children,
    title,
    size = 'xs',
    titleRightContent,
  } = props

  const classes = useStyles()

  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      classes={{ paper: classes.root }}
      maxWidth={size}
    >
      <IconButton className={classes.closeButton} onClick={onClose}>
        <CloseIcon />
      </IconButton>
      <SectionTitle
        text={title}
        tag="h3"
        level={1}
        rightContent={titleRightContent}
        contentClassName={classes.titleContent}
      >
        { props.headerContent }
      </SectionTitle>
      <div className={clsx(classes.content, props.classes?.content)}>{children}</div>
    </Dialog>
  )
}
