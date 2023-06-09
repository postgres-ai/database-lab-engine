import { request } from 'helpers/request'
export const getFullConfig = async () => {
  const response = await request('/admin/config.yaml')
    .then((res) => res.blob())
    .then((blob) => blob.text())
    .then((yamlAsString) => {
      return yamlAsString
    })

  return {
    response: response ? response : null,
    error: response && null,
  }
}
