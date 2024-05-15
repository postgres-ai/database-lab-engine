import React from "react";
import IconButton from "@material-ui/core/IconButton";
import NavigateBeforeIcon from "@material-ui/icons/NavigateBefore";
import NavigateNextIcon from '@material-ui/icons/NavigateNext';
import { Button, makeStyles, useMediaQuery } from "@material-ui/core";
import AddCircleOutlineIcon from "@material-ui/icons/AddCircleOutline";
import Box from "@mui/material/Box";
import { theme } from "@postgres.ai/shared/styles/theme";
import { SettingsWithLabel } from "../SettingsWithLabel/SettingsWithLabel";


export type HeaderButtonsProps = {
  isOpen: boolean;
  onClose: () => void;
  onCreateNewChat: () => void;
  withChatVisibilityButton: boolean;
  onChatVisibilityClick?: () => void;
  currentVisibility: 'public' | 'private';
  permalinkId?: string;
}

const useStyles = makeStyles((theme) => ({
  container: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '8px 5px',
    flex: 1,
    [theme.breakpoints.down('sm')]: {
      justifyContent: 'flex-end',
    }
  },
  hideChatButton: {
    width: '2rem',
    height: '2rem'
  },
  hideChatButtonIcon: {
    width: '2rem',
    height: '2rem',
    fill: '#000'
  },
  createNewChatButton: {
    [theme.breakpoints.down('sm')]: {
      border: 'none',
      minWidth: '2rem',
      height: '2rem',
      padding: 0,
      marginRight: '0.5rem',
      '& .MuiButton-startIcon': {
        margin: 0
      }
    }
  }
}))

export const HeaderButtons = (props: HeaderButtonsProps) => {
  const {
    onClose,
    onCreateNewChat,
    isOpen,
    onChatVisibilityClick,
    withChatVisibilityButton,
    currentVisibility,
    permalinkId
  } = props;
  const matches = useMediaQuery(theme.breakpoints.down('sm'));
  const classes = useStyles();

  return (
    <Box className={classes.container}>
      {
        withChatVisibilityButton && onChatVisibilityClick &&
        <SettingsWithLabel
          chatVisibility={currentVisibility}
          onSettingsClick={onChatVisibilityClick}
          permalinkId={permalinkId}
        />
      }
      <Button
        variant='outlined'
        aria-label="Create new chat"
        startIcon={<AddCircleOutlineIcon />}
        onClick={onCreateNewChat}
        className={classes.createNewChatButton}
      >
        {!matches && 'New Chat'}
      </Button>
      <IconButton
        onClick={onClose}
        aria-label="Hide chats list"
        title={isOpen ? "Hide chats list" : "Open chats list"}
        classes={{root: classes.hideChatButton}}
      >
        {isOpen
          ? <NavigateNextIcon className={classes.hideChatButtonIcon} />
          : <NavigateBeforeIcon className={classes.hideChatButtonIcon} />
        }
      </IconButton>

    </Box>
  )
}