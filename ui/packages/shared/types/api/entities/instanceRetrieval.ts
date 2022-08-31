export interface ActivityType {
  user?: string
  query?: string
  duration?: string
  wait_event_type?: string
  wait_event?: string
}

export type InstanceRetrieval = {
  mode: string
  alerts: Object | null
  lastRefresh: string | null
  nextRefresh: string | null
  status: string
  currentJob?: string
  activity: {
    source: ActivityType[] | null
    target: ActivityType[] | null
  }
}