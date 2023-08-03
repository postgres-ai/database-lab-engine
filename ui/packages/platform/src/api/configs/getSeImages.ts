import { request } from 'helpers/request'

export const getSeImages = async ({
  packageGroup,
  platformUrl,
}: {
  packageGroup: string
  platformUrl?: string
}) => {
  const response = await request(
    `/dblab_se_images?package_group=eq.${packageGroup}
    `,
    {},
    platformUrl,
  )

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}
