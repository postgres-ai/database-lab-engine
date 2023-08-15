import moment from 'moment'

export const formatTimestampUtc = (timestamp: moment.MomentInput) => {
  if (!timestamp) {
    return null
  }

  return moment(timestamp).utc().format('YYYY-MM-DD HH:mm:ss UTC')
}
