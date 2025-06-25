import { useFormik } from 'formik'
import * as Yup from 'yup'

export type FormValues = {
  branch: string
  cloneId: string
  snapshotId: string
  dbUser: string
  dbPassword: string
  isProtected: boolean
}

const Schema = Yup.object().shape({
  cloneId: Yup.string()
    .max(63, 'Clone ID cannot exceed 63 characters')
    .test(
      'valid-docker-name',
      'Clone ID must start with a letter or number and can only contain letters, numbers, underscores, periods, and hyphens',
      (value) => {
        if (!value) return true;
        // Docker container name requirements: start with letter/number, contain only ASCII [a-zA-Z0-9_.-]
        const validDockerName = /^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/;
        return validDockerName.test(value);
      }
    ),
  snapshotId: Yup.string().required('Date state time is required'),
  dbUser: Yup.string().required('Database username is required'),
  dbPassword: Yup.string().required('Database password is required'),
  isProtected: Yup.boolean(),
})

export const useForm = (onSubmit: (values: FormValues) => void) => {
  const formik = useFormik<FormValues>({
    initialValues: {
      branch: '',
      cloneId: '',
      snapshotId: '',
      dbUser: '',
      dbPassword: '',
      isProtected: false,
    },
    validationSchema: Schema,
    onSubmit,
    validateOnBlur: false,
    validateOnChange: false,
  })

  return formik
}
