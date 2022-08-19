import { Config } from "../entities/config"

export type GetConfig = () => Promise<{
  response: Config | null
  error: Response | null
}>
