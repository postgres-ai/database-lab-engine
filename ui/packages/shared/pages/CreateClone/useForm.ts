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
  cloneId: Yup.string(),
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
