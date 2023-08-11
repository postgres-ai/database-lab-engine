import { Tooltip } from '@material-ui/core'
import { makeStyles } from '@material-ui/core'
import { formatDistanceToNowStrict } from 'date-fns'

import { Link } from '@postgres.ai/shared/components/Link2'
import { Instance } from '@postgres.ai/shared/types/api/entities/instance'

import { formatTimestampUtc } from './utils'

const useStyles = makeStyles(
  {
    container: {
      maxWidth: '650px',
      width: '100%',
    },
    timeLabel: {
      lineHeight: '16px',
      fontSize: 14,
      cursor: 'pointer',
      flex: '2 1 0',

      '& span': {
        fontWeight: 'normal !important',
      },
    },
    toolTip: {
      fontSize: '10px !important',
      maxWidth: '100%',
    },
    content: {
      margin: '25px 0',
    },
    flexContainer: {
      display: 'flex',
      alignItems: 'center',
      flexDirection: 'row',
      padding: '10px 0',
      borderBottom: '1px solid rgba(224, 224, 224, 1)',

      '& span:first-child': {
        flex: '1 1 0',
        fontWeight: '500',
      },

      '& span': {
        flex: '2 1 0',
        margin: '0',
      },
    },
  },
  { index: 1 },
)

export const InactiveInstance = ({
  org,
  instance,
}: {
  org: string
  instance: Instance | null
}) => {
  const classes = useStyles()

  const getVersionDigits = (str: string | undefined | null) => {
    if (!str) {
      return 'N/A'
    }

    const digits = str.match(/\d+/g)

    if (digits && digits.length > 0) {
      return `${digits[0]}.${digits[1]}.${digits[2]}`
    }
    return ''
  }

  return (
    <div className={classes.container}>
      <div className={classes.flexContainer}>
        <span>Plan</span>
        <span>{instance?.dto.plan || '---'}</span>
      </div>
      <div className={classes.flexContainer}>
        <span>Version</span>
        <span>
          {getVersionDigits(
            instance?.state && (instance?.state?.engine?.version as string),
          )}
        </span>
      </div>
      <div className={classes.flexContainer}>
        <span>Registered</span>
        <span className={classes.timeLabel}>
          <Tooltip
            title={formatTimestampUtc(instance?.createdAt) ?? ''}
            classes={{ tooltip: classes.toolTip }}
          >
            <span>
              {instance?.createdAt &&
                formatDistanceToNowStrict(new Date(instance?.createdAt), {
                  addSuffix: true,
                })}
            </span>
          </Tooltip>
        </span>
      </div>
      {instance?.telemetryLastReportedAt && (
        <div className={classes.flexContainer}>
          <span>Telemetry last reported</span>
          <span className={classes.timeLabel}>
            <Tooltip
              title={
                formatTimestampUtc(instance?.telemetryLastReportedAt) ?? ''
              }
              classes={{ tooltip: classes.toolTip }}
            >
              <span>
                {formatDistanceToNowStrict(
                  new Date(instance?.telemetryLastReportedAt),
                  {
                    addSuffix: true,
                  },
                )}
              </span>
            </Tooltip>
          </span>
        </div>
      )}
      <div className={classes.flexContainer}>
        <span>Instance ID</span>
        <span>
          <span
            style={{
              fontWeight: 'normal',
            }}
          >
            <Link to={`/${org}/instances/${instance?.id}`}>{instance?.id}</Link>
            &nbsp; (self-assigned:{' '}
            {instance?.dto?.selfassigned_instance_id || 'N/A'})
          </span>
        </span>
      </div>
      {instance?.dto.plan !== 'CE' && (
        <div className={classes.flexContainer}>
          <span>Billing data</span>
          <span>
            <Link to={`/${org}/billing`}>Billing data</Link>
          </span>
        </div>
      )}
      <p className={classes.content}>
        To work with this instance, access its UI or API directly. Full
        integration of UI to the Platform is currently available only to the EE
        users. If you have any questions, reach out to{' '}
        <a
          href="https://postgres.ai/contact"
          target="_blank"
          rel="noopener noreferrer"
        >
          the Postgres.ai team
        </a>
        .
      </p>
    </div>
  )
}
