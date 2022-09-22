export interface ActivityType {
  user?: string
  query?: string
  duration?: number | string
  waitEventType?: string
  waitEvent?: string
}

export type InstanceRetrieval = {
  mode: string
  alerts: Object | null
  lastRefresh: string | null
  nextRefresh: string | null
  status: string
  currentJob?: string
  activity: {
    source: ActivityType[]
    target: ActivityType[]
  }
}

const replaceSinglequote = (string?: string) => string?.replace(/'/g, '')

const formatActivity = (activity: ActivityType[]) =>
  activity?.map((item) => {
    return {
      user: replaceSinglequote(item.user),
      query: replaceSinglequote(item.query),
      duration: replaceSinglequote(`${item.duration}ms`),
      'wait type/event': replaceSinglequote(
        `${item.waitEventType}/${item.waitEvent}`,
      ),
    }
  })

export const formatInstanceRetrieval = (retrieval: InstanceRetrieval) => {
  return {
    ...retrieval,
    activity: {
      source: formatActivity(retrieval.activity?.source),
      target: formatActivity(retrieval.activity?.target),
    },
  }
}

export type InstanceRetrievalType = ReturnType<typeof formatInstanceRetrieval>
