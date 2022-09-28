/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import IconButton from '@material-ui/core/IconButton';
import TextField from '@material-ui/core/TextField';
import Dialog from '@material-ui/core/Dialog';
import MuiDialogTitle from '@material-ui/core/DialogTitle';
import MuiDialogContent from '@material-ui/core/DialogContent';
import MuiDialogActions from '@material-ui/core/DialogActions';
import Typography from '@material-ui/core/Typography';
import Radio from '@material-ui/core/Radio';
import RadioGroup from '@material-ui/core/RadioGroup';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Button from '@material-ui/core/Button';

import { colors } from '@postgres.ai/shared/styles/colors';
import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';
import { Spinner } from '@postgres.ai/shared/components/Spinner';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Urls from '../utils/urls';

const getStyles = (theme) => ({
  textField: {
    ...styles.inputField,
    marginTop: '0px',
    width: 480
  },
  copyButton: {
    marginTop: '-3px',
    fontSize: '20px'
  },
  closeButton: {
    position: 'absolute',
    right: theme.spacing(1),
    top: theme.spacing(1),
    color: theme.palette.grey[500]
  },
  dialog: {
  },
  dialogTitle: {
    fontSize: 16,
    lineHeight: '19px',
    fontWeight: 600
  },
  remark: {
    fontSize: 12,
    lineHeight: '12px',
    color: colors.state.warning,
    paddingLeft: 20
  },
  remarkIcon: {
    display: 'block',
    height: '20px',
    width: '22px',
    float: 'left',
    paddingTop: '5px'
  },
  urlContainer: {
    marginTop: 10,
    paddingLeft: 22
  },
  radioLabel: {
    fontSize: 12
  },
  dialogContent: {
    paddingTop: 10
  }
});

const DialogTitle = withStyles(getStyles)((props) => {
  const { children, classes, onClose, ...other } = props;
  return (
    <MuiDialogTitle disableTypography className={classes.root} {...other}>
      <Typography className={classes.dialogTitle}>{children}</Typography>
      {onClose ? (
        <IconButton aria-label='close' className={classes.closeButton} onClick={onClose}>
          {icons.closeIcon}
        </IconButton>
      ) : null}
    </MuiDialogTitle>
  );
});

const DialogContent = withStyles((theme) => ({
  root: {
    padding: theme.spacing(2)
  }
}))(MuiDialogContent);

const DialogActions = withStyles((theme) => ({
  root: {
    margin: 0,
    padding: theme.spacing(1)
  }
}))(MuiDialogActions);

class ShareUrlDialog extends Component {
  state = {
    shared: null,
    uuid: null
  };

  componentDidMount() {
    const that = this;
    // const { url_type, url_id } = this.props;

    this.unsubscribe = Store.listen(function () {
      let stateData = { data: this.data };

      if (this.data.shareUrl.isAdding) {
        return;
      }

      if (this.data.shareUrl.data && this.data.shareUrl.data.uuid) {
        stateData.shared = 'shared';
        stateData.uuid = this.data.shareUrl.data.uuid;
      } else {
        stateData.shared = 'default';
        stateData.uuid = this.data.shareUrl.uuid;
      }

      that.setState(stateData);
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  copyUrl = () => {
    let copyText = document.getElementById('sharedUrl');

    copyText.select();
    copyText.setSelectionRange(0, 99999);
    document.execCommand('copy');
  };

  closeShareDialog = (close, save) => {
    Actions.closeShareUrlDialog(close, save, this.state.shared === 'shared');
    if (close) {
      this.setState({ data: null, shared: null });
    }
  };

  handleChange = (event) => {
    this.setState({ shared: event.target.value });
  };

  render() {
    const { classes } = this.props;
    const shareUrl = this.state && this.state.data &&
      this.state.data.shareUrl ? this.state.data.shareUrl : null;

    if (!shareUrl ||
      (shareUrl && !shareUrl.isProcessed) ||
      (shareUrl && !shareUrl.open && !shareUrl.data) ||
      this.state.shared === null) {
      return null;
    }

    const urlField = (
      <div>
        <TextField
          id='token'
          className={classes.textField}
          margin='normal'
          value={ Urls.linkShared(this.state.uuid) }
          variant='outlined'
          style={{ width: 500 }}
          InputProps={{
            readOnly: true,
            id: 'sharedUrl'
          }}
          InputLabelProps={{
            shrink: true,
            style: styles.inputFieldLabel
          }}
          FormHelperTextProps={{
            style: styles.inputFieldHelper
          }}
        />

        <IconButton
          className={classes.copyButton}
          aria-label='Copy'
          onClick={this.copyUrl}>
          {icons.copyIcon}
        </IconButton>
      </div>
    );

    return (
      <Dialog
        onClose={() => this.closeShareDialog(true, false)}
        aria-labelledby='customized-dialog-title'
        open={shareUrl.open}
        className={classes.dialog}
      >
        <DialogTitle
          id='customized-dialog-title'
          onClose={() => this.closeShareDialog(true, false)}
        >
          Share
        </DialogTitle>
        <DialogContent dividers className={classes.dialogContent}>
          <RadioGroup
            aria-label='shareUrl'
            name='shareUrl'
            value={this.state.shared}
            onChange={this.handleChange}
            className={classes.radioLabel}
          >
            <FormControlLabel
              value='default'
              control={<Radio />}
              label='Only members of the organization can view'
            />
            <FormControlLabel
              value='shared'
              control={<Radio />}
              label='Anyone with a special link and members of thr organization can view'
            />
          </RadioGroup>
          { shareUrl.remark && <Typography className={classes.remark}>
            <span className={classes.remarkIcon}>
              {icons.warningIcon}
            </span>
            {shareUrl.remark}
          </Typography>}
          {this.state.shared === 'shared' && <div className={classes.urlContainer}>
            {urlField}
          </div>}
        </DialogContent>
        <DialogActions>
          <Button
            autoFocus
            variant='contained'
            disabled={ shareUrl.isRemoving || shareUrl.isAdding }
            onClick={() => this.closeShareDialog(false, true)}
            color='primary'>
            Save changes
            {(shareUrl.isRemoving || shareUrl.isAdding) &&
              <span>
                &nbsp;
                <Spinner size='sm' />
              </span>
            }
          </Button>
          <Button
            onClick={() => this.closeShareDialog(true, false)}
            variant='outlined'
            color='secondary'
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>
    );
  }
}

ShareUrlDialog.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(ShareUrlDialog);
