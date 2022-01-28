export const parseEngineVersion = (str: string) => {
  const [versionStr] = str.split('-')
  const [major, minor, patch] = versionStr.split('.')
  if (!major || !minor || !patch) return null
  return {
    major: Number(major),
    minor: Number(minor),
    patch: Number(patch),
  }
}
