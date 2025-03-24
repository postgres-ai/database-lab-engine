import React, { useMemo, useState } from 'react';
import { TextField, Grid, IconButton } from '@mui/material';
import { Button, makeStyles } from "@material-ui/core";
import { styles } from "@postgres.ai/shared/styles/styles";
import RemoveCircleOutlineIcon from '@material-ui/icons/RemoveCircleOutline';
import { FormikErrors, useFormik } from "formik";
import { FormValues } from "../AuditSettingsForm/AuditSettingsForm";

const useStyles = makeStyles({
  textField: {
    ...styles.inputField,
    maxWidth: 450,
  },
  requestHeadersContainer: {
    paddingTop: '8px!important'
  },
  label: {
    color: '#000!important',
    margin: 0
  },
  requestHeadersTextFieldContainer: {
    flexBasis: 'calc(100% / 2 - 20px)!important',
    width: 'calc(100% / 2 - 20px)!important',
  },
  requestHeadersIconButtonContainer: {
    width: '32px!important',
    height: '32px!important',
    padding: '0!important',
    marginLeft: 'auto!important',
    marginTop: '12px!important',
    '& button': {
      width: 'inherit',
      height: 'inherit'
    }
  }
})

interface SIEMIntegrationFormProps {
  formik: ReturnType<typeof useFormik<FormValues>>;
  disabled: boolean
}

export const SIEMIntegrationForm: React.FC<SIEMIntegrationFormProps> = ({ formik, disabled }) => {
  const classes = useStyles();
  const [isFocused, setIsFocused] = useState(false);
  const [focusedHeaderIndex, setFocusedHeaderIndex] = useState<number | null>(null);

  const getTruncatedUrl = (url: string) => {
    const parts = url.split('/');
    return parts.length > 3 ? parts.slice(0, 3).join('/') + '/*****/' : url;
  };

  const handleHeaderValueDisplay = (index: number, value: string) => {
    if (focusedHeaderIndex === index) {
      return value;
    }
    if (value.length) {
      return "*****";
    } else {
      return ''
    }
  };

  const handleFocusHeaderValue = (index: number) => setFocusedHeaderIndex(index);
  const handleBlurHeaderValue = () => setFocusedHeaderIndex(null);

  const handleFocus = () => setIsFocused(true);
  const handleBlur = () => setIsFocused(false);

  const handleHeaderChange = (index: number, field: 'key' | 'value', value: string) => {
    const headers = formik.values.siemSettings.headers || [];
    const updatedHeaders = [...headers];
    updatedHeaders[index] = {
      ...updatedHeaders[index],
      [field]: value,
    };
    formik.setFieldValue('siemSettings.headers', updatedHeaders);
  };

  const addHeader = () => {
    const headers = formik.values.siemSettings.headers || [];
    const updatedHeaders = [...headers, { key: '', value: '' }];
    formik.setFieldValue('siemSettings.headers', updatedHeaders);
  };

  const removeHeader = (index: number) => {
    const updatedHeaders = formik.values.siemSettings?.headers?.filter((_, i) => i !== index);
    formik.setFieldValue('siemSettings.headers', updatedHeaders);
  };

  return (
    <Grid container spacing={2} maxWidth={450}>
      <Grid item xs={12}>
        <TextField
          id="urlSchemaTextField"
          label="API endpoint"
          variant="outlined"
          className={classes.textField}
          value={isFocused ? formik.values.siemSettings.urlSchema : getTruncatedUrl(formik.values.siemSettings.urlSchema || '')}
          onChange={(e) => formik.setFieldValue('siemSettings.urlSchema', e.target.value)}
          onFocus={handleFocus}
          onBlur={(e) => {
            formik.handleBlur(e);
            handleBlur();
          }}
          margin="normal"
          fullWidth
          placeholder="https://{siem-host}/{path}"
          inputProps={{
            name: 'siemSettings.urlSchema',
            id: 'urlSchemaTextField',
            shrink: 'true',
          }}
          InputLabelProps={{
            shrink: true,
          }}
          disabled={disabled}
          error={formik.touched.siemSettings?.urlSchema && !!formik.errors.siemSettings?.urlSchema}
          helperText={formik.touched.siemSettings?.urlSchema && formik.errors.siemSettings?.urlSchema}
        />
      </Grid>
      <Grid item xs={12} className={classes.requestHeadersContainer}>
        <h3 className={classes.label}>Request headers</h3>
        {formik.values.siemSettings.headers.map((header, index) => (
          <Grid container spacing={1} key={index} alignItems="center">
            <Grid item className={classes.requestHeadersTextFieldContainer}>
              <TextField
                fullWidth
                label="Header Key"
                value={header.key || ''}
                className={classes.textField}
                onChange={(e) => handleHeaderChange(index, 'key', e.target.value)}
                placeholder="Authorization"
                inputProps={{
                  name: `siemSettings.headers[${index}].key`,
                  id: `requestHeaderKeyField${index}`,
                  shrink: 'true',
                }}
                InputLabelProps={{
                  shrink: true,
                }}
                margin="normal"
                disabled={disabled}
              />
            </Grid>
            <Grid item className={classes.requestHeadersTextFieldContainer}>
              <TextField
                fullWidth
                label="Header Value"
                value={handleHeaderValueDisplay(index, header.value || '')}
                className={classes.textField}
                onChange={(e) => handleHeaderChange(index, 'value', e.target.value)}
                onFocus={() => handleFocusHeaderValue(index)}
                onBlur={handleBlurHeaderValue}
                placeholder="token"
                inputProps={{
                  name: `siemSettings.headers[${index}].value`,
                  id: `requestHeaderValueField${index}`,
                  shrink: 'true',
                }}
                InputLabelProps={{
                  shrink: true,
                }}
                margin="normal"
                disabled={disabled}
               />
            </Grid>
            <Grid className={classes.requestHeadersIconButtonContainer} item>
              <IconButton size="small" onClick={() => removeHeader(index)} disabled={disabled}>
                <RemoveCircleOutlineIcon />
              </IconButton>
            </Grid>
          </Grid>
        ))}
        <Button color="primary" variant="outlined" onClick={addHeader} style={{ marginTop: '10px' }} disabled={disabled}>
          Add header
        </Button>
      </Grid>
    </Grid>
  );
};