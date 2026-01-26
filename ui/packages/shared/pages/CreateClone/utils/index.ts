import { FormValues } from '@postgres.ai/shared/pages/CreateClone/useForm'

// escape string for use in single-quoted shell argument
const shellEscape = (str: string): string => {
  // replace single quotes with: end quote, escaped quote, start quote
  return "'" + str.replace(/'/g, "'\\''") + "'"
}

export const getCliCreateCloneCommand = (values: FormValues, showPassword?: boolean) => {
  const { dbUser, dbPassword, branch, protectionDurationMinutes, cloneId } = values

  const usernameDisplay = dbUser ? shellEscape(dbUser) : `<USERNAME>`

  const passwordDisplay = dbPassword
    ? (showPassword ? shellEscape(dbPassword) : dbPassword.replace(/./g, '*'))
    : `<PASSWORD>`

  const cloneIdDisplay = cloneId ? shellEscape(cloneId) : `<CLONE_ID>`

  const protectedFlag = protectionDurationMinutes !== 'none' ? `--protected ${protectionDurationMinutes}` : ''

  return `dblab clone create \

  --username ${usernameDisplay} \

  --password ${passwordDisplay} \

  ${branch ? `--branch ${shellEscape(branch)}` : ``} \

  ${protectedFlag} \

  --id ${cloneIdDisplay} \ `
}

export const getCliCloneStatus = (cloneId: string) => {
  const cloneIdDisplay = cloneId ? shellEscape(cloneId) : `<CLONE_ID>`
  return `dblab clone status ${cloneIdDisplay}`
}
