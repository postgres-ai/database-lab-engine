/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useMemo, useState } from 'react'
import { useFormik } from 'formik'
import * as Yup from 'yup'

import {
  ConnectionStringError,
  connectionStringFromFields,
  connectionStringToFields,
} from './connectionString'

export type RetrievalMode = 'logical' | 'physical'
export type PhysicalTool = 'walg' | 'pgbackrest' | 'customTool' | ''
export type PhysicalEnv = { key: string; value: string }

export type FormValues = {
  debug: boolean
  dockerImage: string
  dockerTag: string
  dockerPath: string
  dockerImageType: string
  sharedBuffers: string
  sharedPreloadLibraries: string
  tuningParams: string
  timetable: string
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
  restoreConfigs: string
  pgDumpCustomOptions: string
  pgRestoreCustomOptions: string
  retrievalMode: RetrievalMode
  physicalTool: PhysicalTool
  physicalDockerImage: string
  physicalSyncEnabled: boolean
  physicalWalgBackupName: string
  physicalPgbackrestStanza: string
  physicalPgbackrestDelta: boolean
  physicalEnvs: PhysicalEnv[]
}

// Logical-mode required fields. Validation skips the source-connection block
// when retrievalMode is "physical" — that branch uses tool-specific inputs
// instead of a connection URL.
const Schema = Yup.object().shape({
  dockerImage: Yup.string().required('Docker image is required'),
  dbname: Yup.string().when('retrievalMode', {
    is: 'logical',
    then: (s) => s.required('Dbname is required'),
  }),
  host: Yup.string().when('retrievalMode', {
    is: 'logical',
    then: (s) => s.required('Host is required'),
  }),
  port: Yup.string().when('retrievalMode', {
    is: 'logical',
    then: (s) => s.required('Port is required'),
  }),
  username: Yup.string().when('retrievalMode', {
    is: 'logical',
    then: (s) => s.required('Username is required'),
  }),
  physicalEnvs: Yup.array()
    .of(
      Yup.object().shape({
        key: Yup.string(),
        value: Yup.string(),
      }),
    )
    .test('unique-keys', '', function (envs) {
      if (!envs) return true

      const positions = new Map<string, number[]>()
      envs.forEach((row, i) => {
        const trimmed = (row?.key ?? '').trim()
        if (!trimmed) return
        const list = positions.get(trimmed) ?? []
        list.push(i)
        positions.set(trimmed, list)
      })

      const dupes: Yup.ValidationError[] = []
      for (const indices of positions.values()) {
        if (indices.length < 2) continue
        for (const i of indices) {
          dupes.push(
            this.createError({
              path: `${this.path}[${i}].key`,
              message: 'Duplicate key',
            }),
          )
        }
      }

      if (dupes.length === 0) return true
      return new Yup.ValidationError(dupes)
    }),
})

export const useForm = (onSubmit: (values: FormValues) => void) => {
  const formik = useFormik<FormValues>({
    initialValues: {
      debug: false,
      dockerImage: '',
      dockerTag: '',
      dockerPath: '',
      dockerImageType: '',
      sharedBuffers: '',
      sharedPreloadLibraries: '',
      tuningParams: '',
      timetable: '',
      dbname: '',
      host: '',
      port: '',
      username: '',
      password: '',
      databases: '',
      dumpParallelJobs: '',
      restoreParallelJobs: '',
      restoreConfigs: '',
      pgDumpCustomOptions: '',
      pgRestoreCustomOptions: '',
      dumpIgnoreErrors: false,
      restoreIgnoreErrors: false,
      retrievalMode: 'logical',
      physicalTool: '',
      physicalDockerImage: '',
      physicalSyncEnabled: false,
      physicalWalgBackupName: '',
      physicalPgbackrestStanza: '',
      physicalPgbackrestDelta: false,
      physicalEnvs: [],
    },
    validationSchema: Schema,
    onSubmit,
    validateOnBlur: false,
    validateOnChange: false,
  })

  const [originalPortWasUnset, setOriginalPortWasUnset] = useState(false)
  const [portDirty, setPortDirty] = useState(false)
  const [connectionStringError, setConnectionStringError] = useState<string | null>(null)

  const connectionString = useMemo(
    () =>
      connectionStringFromFields(
        {
          host: formik.values.host,
          port: formik.values.port || '5432',
          username: formik.values.username,
          dbname: formik.values.dbname,
        },
        { omitDefaultPort: originalPortWasUnset && !portDirty },
      ),
    [
      formik.values.host,
      formik.values.port,
      formik.values.username,
      formik.values.dbname,
      originalPortWasUnset,
      portDirty,
    ],
  )

  const onConnectionStringChange = (s: string) => {
    if (!s.trim()) {
      setConnectionStringError(null)
      formik.setValues({
        ...formik.values,
        host: '',
        port: '',
        username: '',
        dbname: '',
      })
      return
    }

    try {
      const parsed = connectionStringToFields(s)
      setConnectionStringError(null)

      if (parsed.portWasExplicit) setPortDirty(true)

      formik.setValues({
        ...formik.values,
        host: parsed.fields.host,
        port: parsed.fields.port,
        username: parsed.fields.username,
        dbname: parsed.fields.dbname,
      })
    } catch (err) {
      if (err instanceof ConnectionStringError) setConnectionStringError(err.message)
      else throw err
    }
  }

  const markPortInitialState = (wasUnset: boolean) => setOriginalPortWasUnset(wasUnset)
  const markPortDirty = () => setPortDirty(true)
  const omitPortOnSubmit = originalPortWasUnset && !portDirty

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

  return [
    {
      formik,
      connectionData,
      isConnectionDataValid,
      connectionString,
      connectionStringError,
      onConnectionStringChange,
      markPortInitialState,
      markPortDirty,
      omitPortOnSubmit,
    },
  ]
}
