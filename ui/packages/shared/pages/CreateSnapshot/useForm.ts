/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useFormik } from 'formik'
import * as Yup from 'yup'

export type FormValues = {
  cloneID: string
  comment?: string
}

const Schema = Yup.object().shape({
  cloneID: Yup.string().required('Clone ID is required'),
})

export const useForm = (onSubmit: (values: FormValues) => void) => {
  const formik = useFormik<FormValues>({
    initialValues: {
      cloneID: '',
      comment: '',
    },
    validationSchema: Schema,
    onSubmit,
    validateOnBlur: false,
    validateOnChange: false,
  })

  return [{ formik }]
}
