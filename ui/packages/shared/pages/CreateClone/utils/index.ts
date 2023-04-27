import { FormValues } from '@postgres.ai/shared/pages/CreateClone/useForm'

export const getCliCreateCloneCommand = (values: FormValues) => {
  const { dbUser, dbPassword, branch, isProtected, cloneId } = values

  return `dblab clone create \

  --username ${dbUser ? dbUser : `<USERNAME>`} \

  --password ${dbPassword ? dbPassword : `<PASSWORD>`} \

  ${branch ? `--branch ${branch}` : ``} \

  ${isProtected ? `--protected` : ''} \ 

  --id ${cloneId ? cloneId : `<CLONE_ID>`} \ `
}

export const getCliCloneStatus = (cloneId: string) => {
  return `dblab clone status ${cloneId ? cloneId : `<CLONE_ID>`}`
}
