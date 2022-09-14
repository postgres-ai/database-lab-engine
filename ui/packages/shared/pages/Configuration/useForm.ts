/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useFormik } from 'formik'
import * as Yup from 'yup'

export type FormValues = {
  debug: boolean
  dockerImage: string
  sharedBuffers: string
  sharedPreloadLibraries: string
  timetable: string
  dbname: string
  host: string
  port: string
  username: string
  password: string
  databases: string
  pg_dump: string
  pg_restore: string
}

const Schema = Yup.object().shape({
  dockerImage: Yup.string().required('Docker image is required'),
  dbname: Yup.string().required('Dbname is required'),
  host: Yup.string().required('Host is required'),
  port: Yup.string().required('Port is required'),
  username: Yup.string().required('Username is required'),
})

export const useForm = (onSubmit: (values: FormValues) => void) => {
  const formik = useFormik<FormValues>({
    initialValues: {
      debug: false,
      dockerImage: '',
      sharedBuffers: '',
      sharedPreloadLibraries: '',
      timetable: '',
      dbname: '',
      host: '',
      port: '',
      username: '',
      password: '',
      databases: '',
      pg_dump: '',
      pg_restore: '',
    },
    validationSchema: Schema,
    onSubmit,
    validateOnBlur: false,
    validateOnChange: false,
  })

  const formatDatabaseArray = (database: any) => {
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
    ...(formik.values.databases && { db_list: formatDatabaseArray(formik.values.databases)}),
  }

  const isConnectionDataValid =
    formik.values.host &&
    formik.values.port &&
    formik.values.username &&
    formik.values.dbname

  return [{ formik, connectionData, isConnectionDataValid }]
}
