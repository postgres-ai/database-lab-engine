/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useFormik } from 'formik'
import * as Yup from 'yup'
import { WebhookHook } from '@postgres.ai/shared/types/api/entities/config'

export type FormValues = {
  // Global settings
  debug: boolean
  globalDbUsername: string
  globalDbName: string

  // Server settings
  serverHost: string
  serverPort: string

  // Provision settings
  portPoolFrom: string
  portPoolTo: string
  keepUserPasswords: boolean
  cloneAccessAddresses: string

  // Docker image settings
  dockerImage: string
  dockerTag: string
  dockerPath: string
  dockerImageType: string
  sharedBuffers: string
  sharedPreloadLibraries: string
  tuningParams: string

  // Cloning settings
  maxIdleMinutes: string
  accessHost: string

  // Retrieval settings
  timetable: string
  skipStartRefresh: boolean
  dbname: string
  host: string
  port: string
  username: string
  password: string
  databases: string
  dumpParallelJobs: string
  dumpIgnoreErrors: boolean
  restoreParallelJobs: string
  restoreIgnoreErrors: boolean
  pgDumpCustomOptions: string
  pgRestoreCustomOptions: string

  // Diagnostic settings
  logsRetentionDays: string

  // Embedded UI settings
  embeddedUIEnabled: boolean
  embeddedUIDockerImage: string
  embeddedUIHost: string
  embeddedUIPort: string

  // Platform settings
  platformUrl: string
  platformProjectName: string
  platformEnablePersonalToken: boolean
  platformEnableTelemetry: boolean

  // Webhooks settings
  webhooksHooks: WebhookHook[]
}

const Schema = Yup.object().shape({
  dockerImage: Yup.string().required('Docker image is required'),
  dbname: Yup.string().when('$isLogicalMode', {
    is: true,
    then: (schema) => schema.required('Dbname is required'),
    otherwise: (schema) => schema,
  }),
  host: Yup.string().when('$isLogicalMode', {
    is: true,
    then: (schema) => schema.required('Host is required'),
    otherwise: (schema) => schema,
  }),
  port: Yup.string().when('$isLogicalMode', {
    is: true,
    then: (schema) => schema.required('Port is required'),
    otherwise: (schema) => schema,
  }),
  username: Yup.string().when('$isLogicalMode', {
    is: true,
    then: (schema) => schema.required('Username is required'),
    otherwise: (schema) => schema,
  }),
  portPoolFrom: Yup.number().min(1, 'Port must be at least 1').max(65535, 'Port must be at most 65535'),
  portPoolTo: Yup.number().min(1, 'Port must be at least 1').max(65535, 'Port must be at most 65535'),
  serverPort: Yup.number().min(1, 'Port must be at least 1').max(65535, 'Port must be at most 65535'),
  embeddedUIPort: Yup.number().min(1, 'Port must be at least 1').max(65535, 'Port must be at most 65535'),
  logsRetentionDays: Yup.number().min(0, 'Days must be at least 0'),
  maxIdleMinutes: Yup.number().min(0, 'Minutes must be at least 0'),
})

export const useForm = (onSubmit: (values: FormValues) => void) => {
  const formik = useFormik<FormValues>({
    initialValues: {
      // Global settings
      debug: false,
      globalDbUsername: '',
      globalDbName: '',

      // Server settings
      serverHost: '',
      serverPort: '',

      // Provision settings
      portPoolFrom: '',
      portPoolTo: '',
      keepUserPasswords: false,
      cloneAccessAddresses: '',

      // Docker image settings
      dockerImage: '',
      dockerTag: '',
      dockerPath: '',
      dockerImageType: '',
      sharedBuffers: '',
      sharedPreloadLibraries: '',
      tuningParams: '',

      // Cloning settings
      maxIdleMinutes: '',
      accessHost: '',

      // Retrieval settings
      timetable: '',
      skipStartRefresh: false,
      dbname: '',
      host: '',
      port: '',
      username: '',
      password: '',
      databases: '',
      dumpParallelJobs: '',
      restoreParallelJobs: '',
      pgDumpCustomOptions: '',
      pgRestoreCustomOptions: '',
      dumpIgnoreErrors: false,
      restoreIgnoreErrors: false,

      // Diagnostic settings
      logsRetentionDays: '',

      // Embedded UI settings
      embeddedUIEnabled: false,
      embeddedUIDockerImage: '',
      embeddedUIHost: '',
      embeddedUIPort: '',

      // Platform settings
      platformUrl: '',
      platformProjectName: '',
      platformEnablePersonalToken: false,
      platformEnableTelemetry: false,

      // Webhooks settings
      webhooksHooks: [],
    },
    validationSchema: Schema,
    onSubmit,
    validateOnBlur: false,
    validateOnChange: false,
  })

  const formatDatabaseArray = (database: string) => {
    let databases = []
    const splitDatabaseArray = database.split(/[,(\s)(\n)(\r)(\t)(\r\n)]/)

    for (let i = 0; i < splitDatabaseArray.length; i++) {
      if (splitDatabaseArray[i] !== '') {
        databases.push(splitDatabaseArray[i])
      }
    }

    return databases
  }

  const connectionData = {
    host: formik.values.host,
    port: formik.values.port,
    username: formik.values.username,
    password: formik.values.password,
    dbname: formik.values.dbname,
    ...(formik.values.databases && {
      db_list: formatDatabaseArray(formik.values.databases),
    }),
    ...(formik.values.dockerImageType === 'custom' && {
      dockerImage: formik.values.dockerImage,
    }),
  }

  const isConnectionDataValid =
    formik.values.host &&
    formik.values.port &&
    formik.values.username &&
    formik.values.dbname

  return [{ formik, connectionData, isConnectionDataValid }]
}
