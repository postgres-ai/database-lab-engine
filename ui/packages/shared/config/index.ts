import { Locale } from 'date-fns'
import { enUS } from 'date-fns/locale'
import { getUserLocale } from 'get-user-locale'

type Config = {
  dateFnsLocale: Locale
  appName: string
}

export const config: Config = {
  dateFnsLocale: enUS,
  appName: 'Postgres.AI',
}

const loadDateFnsLocale = async () => {
  const userLocale = getUserLocale()

  // We are already using this locale.
  if (userLocale === config.dateFnsLocale.code) return

  try {
    const locale = await import(`date-fns/locale/${userLocale}`)
    config.dateFnsLocale = locale.default
    return
  } catch (e) {
    // Unavailable locale.
  }
}

export const initConfig = async () => {
  await loadDateFnsLocale()
}
