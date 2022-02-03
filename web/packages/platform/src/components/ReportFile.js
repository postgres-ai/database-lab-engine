/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import TextField from '@material-ui/core/TextField';
import ReactMarkdown from 'react-markdown';
import Button from '@material-ui/core/Button';

import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { styles } from '@postgres.ai/shared/styles/styles';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Error from './Error';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import Urls from '../utils/urls';


const getStyles = theme => ({
  root: {
    width: '100%',
    [theme.breakpoints.down('sm')]: {
      maxWidth: '100vw'
    },
    [theme.breakpoints.up('md')]: {
      maxWidth: 'calc(100vw - 240px)'
    },
    [theme.breakpoints.up('lg')]: {
      maxWidth: 'calc(100vw - 240px)'
    },
    minHeight: '100%',
    zIndex: 1,
    position: 'relative'
  },
  reportFileContent: {
    border: '1px solid silver',
    margin: 5,
    padding: 5
  },
  code: {
    'width': '100%',
    'background-color': 'rgb(246, 248, 250)',
    '& > div > textarea': {
      fontFamily: '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
        ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
      color: 'black',
      fontSize: 14
    }
  },
  rawCode: {
    'width': '100%',
    'background-color': 'none',
    'margin-top': '10px',
    'padding': '0px',
    '& > .MuiOutlinedInput-multiline': {
      padding: '0px!important'
    },
    '& > div > textarea': {
      fontFamily: '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
        ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
      color: 'black',
      fontSize: 14
    },
    '& > .MuiInputBase-fullWidth > fieldset': {
      borderWidth: 'none!important',
      borderStyle: 'none!important',
      borderRadius: '0px!important'
    }
  },
  rawProgress: {
    position: 'absolute',
    left: 'calc( 100vw/2 - 32px )',
    top: 'calc( 100vh/2 - 32px )'
  },
  bottomSpace: {
    ...styles.bottomSpace
  }
});

const mdViewerStyles = (
  <style>{`
    .md-report-file-preview{
        margin: 5px;
        font-size: 14px;
    }
    .md-report-file-preview h1 {
        margin-top: 5px;
    }

    .md-report-file-preview table{
        border-collapse: collapse;
        border-spacing: 0px;
    }

    .md-report-file-preview pre {
        border: 1px solid #ccc;
        background: #f6f8fa;
        padding: 5px;
    }

    .md-report-file-preview blockquote {
        color: #666;
        margin: 0;
        padding-left: 3em;
        border-left: 0.5em #eee solid;
    }

    .md-report-file-preview tr {
        border-top: 1px solid #c6cbd1;
        background: #fff;
    }

    .md-report-file-preview th,
    .md-report-file-preview td {
        padding: 10px 13px;
        border: 1px solid #dfe2e5;
    }

    .md-report-file-preview table tr:nth-child(2n) {
        background: #f6f8fa;
    }
    .md-report-file-preview img.emoji{
        margin-top: 5px;
    }
`}
  </style>
);

const textAreaStyles = (
  <style>{`
    textarea {
      -moz-tab-size: 4;
      -webkit-tab-size: 4;
      -o-tab-size : 4;
      tab-size : 4;
    }
`}
  </style>
);

class ReportFile extends Component {
  getFileId() {
    let id = parseInt(this.props.match.params.fileId, 10);
    /* eslint eqeqeq: 1 */
    id = (id == this.props.match.params.fileId) ? id :
      this.props.match.params.fileId;
    /* eslint eqeqeq: 0 */

    return id;
  }

  componentDidMount() {
    const that = this;
    const projectId = this.props.projectId;
    const reportId = parseInt(this.props.match.params.reportId, 10);
    const type = this.props.match.params.fileType;
    const id = this.getFileId();
    const fileId = type.toLowerCase() + '_' + id;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const reportFile = this.data && this.data.reportFile ?
        this.data.reportFile : null;
      const report = this.data && this.data.report ?
        this.data.report : null;

      that.setState({ data: this.data });

      if (auth && auth.token && !reportFile.files[fileId] &&
        !reportFile.isProcessing && !reportFile.error) {
        Actions.getCheckupReportFile(auth.token, projectId, reportId, id, type);
      }

      if (auth && auth.token && !report.isProcessing && !report.data) {
        Actions.getCheckupReportFiles(auth.token, reportId, 'md',
          'filename', 'asc');
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  downloadFile(fileId) {
    let type = this.props.match.params.fileType;
    const data = this.state.data && this.state.data.reportFile ?
      this.state.data.reportFile : null;
    const fileName = data.files[fileId].filename;
    let content = data.files[fileId].text ?
      data.files[fileId].text : data.files[fileId].data;

    if (type === 'json') {
      let jsonData = null;
      try {
        jsonData = JSON.parse(content);
        content = JSON.stringify(jsonData, null, '\t');
      } catch (err) {
        content = false;
        console.log(err);
      }
    }

    if (content) {
      let a = document.createElement('a');
      let file = new Blob([content], { type: 'application/json' });
      a.href = URL.createObjectURL(file);
      a.download = fileName;
      a.click();
    }

    return false;
  }

  markdownLink(linkProps) {
    const { href, target, children } = linkProps;
    const reportId = this.props.match.params.reportId;
    let fileName = null;

    let match = href.match(/..\/..\/json_reports\/(.*)\/K_query_groups\/(.*)/);
    if (match && match.length > 2) {
      fileName = match[2];
    }

    if (fileName) {
      return (
        <a
          href={Urls.linkReportQueryFull(this.props, reportId, fileName)}
          target={'_blank'}
        >
          {children}
        </a>
      );
    }

    if (!href.startsWith('#')) {
      // Show external link in new tab
      return (
        <a href={href} target={'_blank'}>{children}</a>
      );
    }

    return (
      <a href={href} target={target}>{children}</a>
    );
  }

  copyFile() {
    let copyText = document.getElementById('fileContent');

    copyText.select();
    copyText.setSelectionRange(0, 99999);
    document.execCommand('copy');
    copyText.setSelectionRange(0, 0);
  }

  render() {
    const that = this;
    const { classes } = this.props;
    const data = this.state && this.state.data && this.state.data.reportFile ?
      this.state.data.reportFile : null;
    const reportId = this.props.match.params.reportId;
    const id = this.getFileId();
    const type = this.props.match.params.fileType;
    const fileId = type.toLowerCase() + '_' + id;
    const raw = this.props.raw;

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        {...this.props}
        breadcrumbs={[
          { name: 'Reports', url: 'reports' },
          { name: 'Report #' + reportId, url: Urls.linkReport(this.props, reportId, type) },
          { name: data && data.files && data.files[fileId] ?
            data.files[fileId].filename : 'Report file' }
        ]}
      />
    );

    if (this.state && this.state.data && this.state.data.reportFile.error) {
      return (
        <div>
          {!raw && breadcrumbs}

          <Error
            message={this.state.data.reportFile.errorMessage}
            code={this.state.data.reportFile.errorCode}
          />
        </div>
      );
    }

    if (!data || (data && data.isProcessing) || (data && !data.files[fileId])) {
      return (
        <>
          {!raw && breadcrumbs}

          <PageSpinner className={!raw ? classes.progress : classes.rawProgress} />
        </>
      );
    }

    let fileContent = null;
    let fileLink = null;
    let copyBtn = null;
    if (type === 'md') {
      fileContent = (
        <div>
          {mdViewerStyles}

          <ReactMarkdown
            className='md-report-file-preview'
            source={data.files[fileId].data}
            escapeHtml={false}
            renderers={{
              link: (props) => {
                return that.markdownLink(props);
              }
            }}
          />
        </div>
      );
    } else {
      let content = data.files[fileId].text ? data.files[fileId].text :
        data.files[fileId].data;

      if (type === 'json') {
        let jsonData = null;

        try {
          jsonData = JSON.parse(content);
          content = JSON.stringify(jsonData, null, '\t');
        } catch (err) {
          console.log(err);
        }
      }

      fileContent = (
        <div>
          {textAreaStyles}

          <TextField
            id='fileContent'
            multiline
            fullWidth
            value={content}
            className={!raw ? classes.code : classes.rawCode}
            variant='outlined'
            margin='normal'
            InputProps={{
              readOnly: true,
              id: 'fileContent'
            }}
          />
        </div>
      );

      fileLink = (
        <Button
          variant='contained'
          color='primary'
          onClick={() => this.downloadFile(fileId)}
        >
          Download
        </Button>
      );

      copyBtn = (
        <Button
          variant='contained'
          color='primary'
          style={{ marginLeft: 10 }}
          onClick={() => this.copyFile()}
        >
          Copy
        </Button>
      );
    }

    return (
      <div
        className={classes.root}
        style={raw ? {
          'width': '100vw',
          'height': '100vh',
          'max-width': '100vw',
          'max-height': '100vh',
          'min-width': '100vw',
          'min-height': '100vh',
          'overflow': 'scroll'
        } : {}}
      >
        <div style={raw ? { padding: '10px' } : {}}>
          {!raw && breadcrumbs}

          {fileLink}
          {raw && copyBtn}

          {fileContent}
        </div>

        <div className={classes.bottomSpace}/>
      </div>
    );
  }
}

ReportFile.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(ReportFile);
