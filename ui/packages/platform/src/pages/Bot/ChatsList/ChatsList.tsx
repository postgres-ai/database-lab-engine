/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from "react";
import { Link } from "react-router-dom";
import { useParams } from "react-router";
import cn from "classnames";
import { ListItem, ListItemIcon, makeStyles, Theme, useMediaQuery } from "@material-ui/core";
import Drawer from '@material-ui/core/Drawer';
import List from "@material-ui/core/List";
import Divider from "@material-ui/core/Divider";
import ListSubheader from '@material-ui/core/ListSubheader';
import Box from "@mui/material/Box";
import { Spinner } from "@postgres.ai/shared/components/Spinner";
import { HeaderButtons, HeaderButtonsProps } from "../HeaderButtons/HeaderButtons";
import { theme } from "@postgres.ai/shared/styles/theme";
import { useAiBot } from "../hooks";
import VisibilityOffIcon from '@material-ui/icons/VisibilityOff';


const useStyles = makeStyles<Theme, ChatsListProps>((theme) => ({
    drawerPaper: {
      width: 240,
      //TODO: Fix magic numbers
      height: props => props.isDemoOrg ? 'calc(100vh - 122px)' : 'calc(100vh - 90px)',
      marginTop: props => props.isDemoOrg ? 72 : 40,
      [theme.breakpoints.down('sm')]: {
        height: '100vh!important',
        marginTop: '0!important',
        width: 320,
        zIndex: 9999
      },
      '& > ul': {
        display: 'flex',
        flexDirection: 'column',
        '@supports (scrollbar-gutter: stable)': {
          scrollbarGutter: 'stable',
          paddingRight: 0,
          overflow: 'hidden',
        },
        '&:hover': {
          overflow: 'auto'
        },
        [theme.breakpoints.down('sm')]: {
          paddingBottom: 120
        }
      }
    },
    listPadding: {
      paddingTop: 0
    },
    listSubheaderRoot: {
      background: 'white',
      [theme.breakpoints.down('sm')]: {
        padding: 0
      }
    },
    listItemLink: {
      fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
      fontStyle: 'normal',
      fontWeight: 'normal',
      fontSize: '0.875rem',
      lineHeight: '1rem',
      color: '#000000',
      width: '100%',
      textOverflow: 'ellipsis',
      overflow: 'hidden',
      padding: '0.75rem 1rem',
      whiteSpace: 'nowrap',
      textDecoration: "none",
      flex: '0 0 2.5rem',
      display: 'block',
      '&:hover': {
        background: 'rgba(0, 0, 0, 0.04)'
      },
      '&:focus': {
        outline: 'none',
        background: 'rgba(0, 0, 0, 0.04)'
      }
    },
    listItemLinkActive: {
      background: 'rgba(0, 0, 0, 0.04)'
    },
    listItemIcon: {
      transform: 'translateY(2px)',
      marginRight: 2,
      minWidth: 'auto',
    },
    loader: {
      width: '100%',
      height: '100%',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center'
    }
  })
);

type ChatsListProps = {
  isOpen: boolean;
  onCreateNewChat: () => void;
  onClose: () => void;
  isDemoOrg: boolean;
  onLinkClick?: (targetThreadId: string) => void;
} & HeaderButtonsProps

export const ChatsList = (props: ChatsListProps) => {
  const {
    isOpen,
    onCreateNewChat,
    onClose,
    withChatVisibilityButton,
    onSettingsClick,
    onLinkClick,
    onConsoleClick
  } = props;

  const { chatsList, chatsListLoading: loading } = useAiBot();

  const classes = useStyles(props);
  const params = useParams<{ org?: string, threadId?: string }>();
  const matches = useMediaQuery(theme.breakpoints.down('sm'));
  const linkBuilder = (msgId: string) => {
    if (params.org) {
      return `/${params.org}/bot/${msgId}`
    } else {
      return `/bot/${msgId}`
    }
  }

  const handleClick = (threadId: string) => {
    if (onLinkClick) {
      onLinkClick(threadId)
      if (matches) {
        onClose()
      }
    }
  }

  const handleCloseOnClickOutside = () => {
    if (matches) {
      onClose()
    }
  }

  const loader = (
    <Box className={classes.loader}>
      <Spinner/>
    </Box>
  )

  const list = (
    <List
      classes={{padding: classes.listPadding}}
    >
      <ListSubheader
        classes={{root: classes.listSubheaderRoot}}
      >
        <HeaderButtons
          onClose={onClose}
          onCreateNewChat={onCreateNewChat}
          isOpen={isOpen}
          withChatVisibilityButton={withChatVisibilityButton}
          onSettingsClick={onSettingsClick}
          onConsoleClick={onConsoleClick}
        />
        <Divider/>
      </ListSubheader>

      {chatsList && chatsList.map((item) => {
        const isActive = item.id === params.threadId
        const link = linkBuilder(item.id)
        return (
          <ListItem
            component={Link}
            to={link}
            key={item.id}
            className={cn(classes.listItemLink, {[classes.listItemLinkActive]: isActive})}
            id={item.id}
            onClick={() => handleClick(item.id)}
            autoFocus={isActive}
          >
            <ListItemIcon
              className={classes.listItemIcon}
              title={item.is_public ? 'This thread is public' : 'This thread is private'}>
              {!item.is_public && <VisibilityOffIcon />}
            </ListItemIcon>
            {item.content}
          </ListItem>
        )
      })
      }
    </List>
  )

  return (
    <Drawer
      variant={matches ? 'temporary' : 'persistent'}
      anchor="right"
      BackdropProps={{ invisible: true }}
      elevation={1}
      open={isOpen}
      onClose={handleCloseOnClickOutside}
      classes={{
        paper: classes.drawerPaper
      }}
    >
      {loading ? loader : list}
    </Drawer>
  )
}