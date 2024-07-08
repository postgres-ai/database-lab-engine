import React from "react";
import { Button, makeStyles, useMediaQuery } from "@material-ui/core";
import IconButton from "@material-ui/core/IconButton";
import NavigateBeforeIcon from "@material-ui/icons/NavigateBefore";
import NavigateNextIcon from '@material-ui/icons/NavigateNext';
import AddCircleOutlineIcon from "@material-ui/icons/AddCircleOutline";
import Box from "@mui/material/Box";
import { theme } from "@postgres.ai/shared/styles/theme";
import { SettingsPanel, SettingsPanelProps } from "../SettingsPanel/SettingsPanel";


export type HeaderButtonsProps = {
  isOpen: boolean;
  onClose: () => void;
  onCreateNewChat: () => void;
  withChatVisibilityButton: boolean;
  onSettingsClick: SettingsPanelProps["onSettingsClick"];
  onConsoleClick: SettingsPanelProps["onConsoleClick"];
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
      marginRight: '0.25rem',
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
    onSettingsClick,
    withChatVisibilityButton,
    onConsoleClick
  } = props;
  const matches = useMediaQuery(theme.breakpoints.down('sm'));
  const classes = useStyles();


  return (
    <Box className={classes.container}>
      {
        withChatVisibilityButton &&
          <SettingsPanel
            onSettingsClick={onSettingsClick}
            onConsoleClick={onConsoleClick}
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