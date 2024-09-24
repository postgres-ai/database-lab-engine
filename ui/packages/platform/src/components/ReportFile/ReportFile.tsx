/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component, HTMLAttributeAnchorTarget } from 'react'
import { Button, TextField } from '@material-ui/core'
import ReactMarkdown from 'react-markdown'
import rehypeRaw from 'rehype-raw'
import remarkGfm from 'remark-gfm'

import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import Urls from '../../utils/urls'
import { ReportFileProps } from 'components/ReportFile/ReportFileWrapper'

interface LinkProps {
  href: string | HTMLAttributeAnchorTarget
  target: string
  children: React.ReactNode
}

interface ReportFileWithStylesProps extends ReportFileProps {
  classes: ClassesType
}

interface ReportFileState {
  data: {
    auth: {
      token: string
    } | null
    reportFile: {
      isProcessing: boolean
      error: boolean
      errorCode: number
      errorMessage: string
      files: {
        [file: string]: {
          filename: string
          text: string
          data: string
        }
      }
    } | null
    report: {
      isProcessing: boolean
      data: {
        id: number
      }[]
    } | null
  } | null
}

const mdViewerStyles = (
  <style>
    {`
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
)

const textAreaStyles = (
  <style>
    {`
    textarea {
      -moz-tab-size: 4;
      -webkit-tab-size: 4;
      -o-tab-size : 4;
      tab-size : 4;
    }
`}
  </style>
)

class ReportFile extends Component<ReportFileWithStylesProps, ReportFileState> {
  getFileId() {
    let fileID = this.props.match.params.fileId
    let parseID = parseInt(fileID, 10)

    // To distinct different fileIDs. For example, "72215" and "1_1.sql".
    // {ORG_URL}/reports/268/files/72215/md
    // {ORG_URL}/reports/268/files/1_1.sql/sql?raw
    return (fileID.toString() == parseID.toString()) ? parseID : fileID
  }

  componentDidMount() {
    const that = this
    const projectId = this.props.projectId
    const reportId = parseInt(this.props.match.params.reportId, 10)
    const type = this.props.match.params.fileType
    const id = this.getFileId()
    const fileId = type.toLowerCase() + '_' + id

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      const auth = this.data && this.data.auth ? this.data.auth : null
      const reportFile =
        this.data && this.data.reportFile ? this.data.reportFile : null
      const report = this.data && this.data.report ? this.data.report : null

      that.setState({ data: this.data })

      if (
        auth &&
        auth.token &&
        !reportFile?.files[fileId] &&
        !reportFile?.isProcessing &&
        !reportFile?.error
      ) {
        Actions.getCheckupReportFile(auth.token, projectId, reportId, id, type)
      }

      if (auth && auth.token && !report?.isProcessing && !report?.data) {
        Actions.getCheckupReportFiles(
          auth.token,
          reportId,
          'md',
          'filename',
          'asc',
        )
      }
    })

    Actions.refresh()
  }

  unsubscribe: Function
  componentWillUnmount() {
    this.unsubscribe()
  }

  downloadFile(fileId: string) {
    let type = this.props.match.params.fileType
    const data =
      this.state.data && this.state.data.reportFile
        ? this.state.data.reportFile
        : null
    const fileName = data?.files[fileId].filename
    let content = data?.files[fileId].text
      ? data.files[fileId].text
      : (data?.files[fileId].data as string)

    if (type === 'json') {
      let jsonData = null
      try {
        jsonData = JSON.parse(content)
        content = JSON.stringify(jsonData, null, '\t')
      } catch (err) {
        content = ''
        console.log(err)
      }
    }

    if (content) {
      let a: HTMLAnchorElement = document.createElement('a')
      let file = new Blob([content], { type: 'application/json' })
      a.href = URL.createObjectURL(file)
      a.download = fileName as string
      a.click()
    }

    return false
  }

  markdownLink(linkProps: LinkProps) {
    const { href, target, children } = linkProps
    const reportId = this.props.match.params.reportId
    let fileName = null

    let match = href.match(/..\/..\/json_reports\/(.*)\/K_query_groups\/(.*)/)
    if (match && match.length > 2) {
      fileName = match[2]
    }

    if (fileName) {
      return (
        <a
          href={Urls.linkReportQueryFull(this.props, reportId, fileName)}
          target={'_blank'}
          rel="noreferrer"
        >
          {children}
        </a>
      )
    }

    if (!href.startsWith('#')) {
      // Show external link in new tab
      return (
        <a href={href} target={'_blank'} rel="noreferrer">
          {children}
        </a>
      )
    }

    return (
      <a href={href} target={target}>
        {children}
      </a>
    )
  }

  copyFile() {
    let copyText = document.getElementById('fileContent') as HTMLInputElement
    if (copyText) {
      copyText.select()
      copyText.setSelectionRange(0, 99999)
      document.execCommand('copy')
      copyText.setSelectionRange(0, 0)
    }
  }

  render() {
    const that = this
    const { classes } = this.props
    const data =
      this.state && this.state.data && this.state.data.reportFile
        ? this.state.data.reportFile
        : null
    const reportId = this.props.match.params.reportId
    const id = this.getFileId()
    const type = this.props.match.params.fileType
    const fileId: string = type.toLowerCase() + '_' + id
    const raw = this.props.raw

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        {...this.props}
        breadcrumbs={[
          { name: 'Reports', url: 'reports' },
          {
            name: 'Report #' + reportId,
            url: Urls.linkReport(this.props, reportId, type),
          },
          {
            name:
              data && data.files && data.files[fileId]
                ? data.files[fileId].filename
                : 'Report file',
          },
        ]}
      />
    )

    if (this.state && this.state.data && this.state.data?.reportFile?.error) {
      return (
        <div>
          {!raw && breadcrumbs}

          <ErrorWrapper
            message={this.state.data.reportFile.errorMessage}
            code={this.state.data.reportFile.errorCode}
          />
        </div>
      )
    }

    if (!data || (data && data.isProcessing) || (data && !data.files[fileId])) {
      return (
        <>
          {!raw && breadcrumbs}

          <PageSpinner />
        </>
      )
    }

    let fileContent = null
    let fileLink = null
    let copyBtn = null
    if (type === 'md') {
      fileContent = (
        <div>
          {mdViewerStyles}

          <ReactMarkdown
            className="md-report-file-preview"
            children={data.files[fileId].data}
            rehypePlugins={[rehypeRaw]}
            remarkPlugins={[remarkGfm]}
            components={{
              a: (props) => {
                return that.markdownLink(props as LinkProps)
              },
            }}
          />
        </div>
      )
    } else {
      let content = data.files[fileId].text
        ? data.files[fileId].text
        : data.files[fileId].data

      if (type === 'json') {
        let jsonData = null

        try {
          jsonData = JSON.parse(content)
          content = JSON.stringify(jsonData, null, '\t')
        } catch (err) {
          console.log(err)
        }
      }

      fileContent = (
        <div>
          {textAreaStyles}

          <TextField
            id="fileContent"
            multiline
            fullWidth
            value={content}
            className={!raw ? classes.code : classes.rawCode}
            variant="outlined"
            margin="normal"
            InputProps={{
              readOnly: true,
              id: 'fileContent',
            }}
          />
        </div>
      )

      fileLink = (
        <Button
          variant="contained"
          color="primary"
          onClick={() => this.downloadFile(fileId)}
        >
          Download
        </Button>
      )

      copyBtn = (
        <Button
          variant="contained"
          color="primary"
          style={{ marginLeft: 10 }}
          onClick={() => this.copyFile()}
        >
          Copy
        </Button>
      )
    }

    return (
      <div
        className={classes.root}
        style={
          raw
            ? {
                width: '100vw',
                height: '100vh',
                maxWidth: '100vw',
                maxHeight: '100vh',
                minWidth: '100vw',
                minHeight: '100vh',
                overflow: 'scroll',
              }
            : {}
        }
      >
        <div style={raw ? { padding: '10px' } : {}}>
          {!raw && breadcrumbs}

          {fileLink}
          {raw && copyBtn}

          {fileContent}
        </div>

        <div className={classes.bottomSpace} />
      </div>
    )
  }
}

export default ReportFile
