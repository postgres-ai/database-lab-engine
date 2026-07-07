// Shared image-catalog constants. Kept in its own module so both
// `configOptions.ts` and `utils/index.ts` can read them without creating
// an import cycle (utils → configOptions today).

export const genericImagePrefix = 'postgresai/extended-postgres'

// Predefined Docker image catalog for the Generic Postgres image. If a
// user's config references a tag not listed here, createEnhancedDockerImages
// in utils/index.ts appends it at runtime.
export const dockerImagesConfig: Record<string, string[]> = {
  '9.6': ['0.5.3', '0.5.2', '0.5.1'],
  '10': ['0.5.3', '0.5.2', '0.5.1'],
  '11': ['0.5.3', '0.5.2', '0.5.1'],
  '12': ['0.5.3', '0.5.2', '0.5.1'],
  '13': ['0.5.3', '0.5.2', '0.5.1'],
  '14': ['0.5.3', '0.5.2', '0.5.1'],
  '15': ['0.5.3', '0.5.2', '0.5.1'],
  '16': ['0.5.3', '0.5.2', '0.5.1'],
  '17': ['0.5.3', '0.5.2', '0.5.1'],
  '18': ['0.6.1'],
}
