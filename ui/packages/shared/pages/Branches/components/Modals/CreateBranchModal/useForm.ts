/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useFormik } from 'formik'
import { CreateBranchFormValues } from '@postgres.ai/shared/types/api/endpoints/createBranch'

import * as Yup from 'yup'

const Schema = Yup.object().shape({
  branchName: Yup.string().required('Branch name is required'),
})

export const useForm = (onSubmit: (values: CreateBranchFormValues) => void) => {
  const formik = useFormik<CreateBranchFormValues>({
    initialValues: {
      branchName: '',
      baseBranch: 'main',
      snapshotID: '',
      creationType: 'branch',
    },
    validationSchema: Schema,
    onSubmit,
    validateOnBlur: false,
    validateOnChange: false,
  })

  const isFormDisabled =
    formik.isSubmitting ||
    !formik.values.branchName ||
    (!formik.values.snapshotID && !formik.values.baseBranch)

  return [{ formik, isFormDisabled }]
}
