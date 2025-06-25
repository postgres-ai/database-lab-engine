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
  branchName: Yup.string()
    .required('Branch name is required')
    .max(255, 'Branch name cannot exceed 255 characters')
    .test(
      'valid-zfs-name',
      'Branch name can only contain letters, numbers, and these characters: _ -',
      (value) => {
        if (!value) return true;
        // ZFS allows only alphanumeric characters and: _ -
        const validChars = /^[a-zA-Z0-9_-]*$/;
        return validChars.test(value);
      }
    )
    .test(
      'valid-start-char',
      'Branch name must start with a letter, number, or underscore',
      (value) => {
        if (!value) return true;
        // Dataset names can begin with alphanumeric character or underscore
        const validStartChar = /^[a-zA-Z0-9_]/;
        return validStartChar.test(value);
      }
    ),
})

export const useForm = (onSubmit: (values: CreateBranchFormValues) => void) => {
  const formik = useFormik<CreateBranchFormValues>({
    initialValues: {
      instanceId: '',
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
