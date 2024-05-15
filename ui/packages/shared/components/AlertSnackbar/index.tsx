import React from "react";
import Snackbar from '@mui/material/Snackbar';
import Alert from '@mui/material/Alert';
import {useAlertSnackbar} from "./useAlertSnackbar";

export const AlertSnackbar = () => {
  const { snackbarMessage, closeSnackbar } = useAlertSnackbar();
  return (
    <Snackbar
      open={!!snackbarMessage}
      onClose={closeSnackbar}
    >
      <Alert
        onClose={closeSnackbar}
        severity="error"
        variant="filled"
        sx={{ width: '100%' }}
      >
        {snackbarMessage?.message}
      </Alert>
    </Snackbar>
  )
}