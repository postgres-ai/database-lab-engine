import { useState } from 'react'

import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { TextField } from '@postgres.ai/shared/components/TextField'
import { Button } from '@postgres.ai/shared/components/Button'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { localStorage } from 'helpers/localStorage'
import { appStore } from 'stores/app'
import { ROUTES } from 'config/routes'

import { Card } from './Card'

import styles from './styles.module.scss'

export const Auth = () => {
  const [authToken, setAuthToken] = useState('')

  const auth = () => {
    localStorage.setAuthToken(authToken)
    appStore.setIsValidAuthToken()
  }

  return (
    <PageContainer>
      <NavPath routes={[ROUTES, ROUTES.AUTH]} />
      <div className={styles.content}>
        <Card
          className={styles.form}
          onSubmit={(e) => {
            e.preventDefault()
            auth()
          }}
        >
          <SectionTitle tag="h1" level={1} text="Authentication" />
          <p className={styles.desc}>
            Please enter the <strong>verification token</strong> (or keep the
            field empty if authorization is disabled).
          </p>
          <TextField
            className={styles.field}
            value={authToken}
            onChange={(e) => setAuthToken(e.target.value)}
            fullWidth
            label="Verification token"
            type="password"
            placeholder="Verification token"
          />
          <Button
            variant="primary"
            size="large"
            className={styles.button}
            type="submit"
          >
            Auth
          </Button>
        </Card>
      </div>
    </PageContainer>
  )
}
