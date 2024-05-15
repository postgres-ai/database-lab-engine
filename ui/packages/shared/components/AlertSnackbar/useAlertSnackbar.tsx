import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";
import {AlertSnackbar} from "./index";

type AlertSnackbarMessage = {
  message: string,
  key: number
}

type AlertSnackbarContextType = {
  snackbarMessage: AlertSnackbarMessage | null;
  showMessage: (message: string) => void;
  closeSnackbar: () => void;
}

const AlertSnackbarContext = createContext<AlertSnackbarContextType | undefined>(undefined);

export const useAlertSnackbar = () => {
  const context = useContext(AlertSnackbarContext);
  if (context === undefined) {
    throw new Error('useSnackbar must be used within a SnackbarProvider');
  }
  return context;
};

type SnackbarProviderProps = {
  children: React.ReactNode
}

export const AlertSnackbarProvider = (props: SnackbarProviderProps) => {
  const { children } = props;
  const [snackbarMessage, setSnackbarMessage] = useState<AlertSnackbarMessage | null>(null);
  const isMounted = useRef(true);

  useEffect(() => {
    return () => {
      isMounted.current = false;
    };
  }, []);

  const showMessage = useCallback((message: string) => {
    if (isMounted.current) {
      setSnackbarMessage({message, key: new Date().getTime()});
    }
  }, []);

  const closeSnackbar = useCallback(() => {
    if (isMounted.current) {
      setSnackbarMessage(null);
    }
  }, []);

  const value = {
    snackbarMessage,
    showMessage,
    closeSnackbar,
  };

  return (
    <AlertSnackbarContext.Provider value={value}>
      {children}
      <AlertSnackbar />
    </AlertSnackbarContext.Provider>
  );

}