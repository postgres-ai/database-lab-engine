/* eslint-disable no-undef */

Cypress.on('uncaught:exception', () => {
  return false
})

function setupIntercepts() {
  const exceptions = [
    '/healthz',
    '/instance/retrieval',
    '/status',
    '/admin/config',
  ]

  cy.intercept('GET', '/healthz*', {
    statusCode: 200,
    body: {
      edition: 'standard',
    },
  })

  cy.intercept('GET', '/instance/retrieval*', {
    statusCode: 200,
    body: {
      status: 'inactive',
    },
  })

  cy.intercept('GET', '/status*', {
    statusCode: 200,
    body: {
      status: {
        code: 'OK',
        message: 'Instance is ready',
      },
      pools: [],
      cloning: {
        clones: [],
      },
      retrieving: {
        status: 'inactive',
      },
    },
  })

  cy.intercept('GET', '*', (req) => {
    if (
      req.resourceType === 'fetch' &&
      exceptions.every((e) => !req.url.includes(e))
    ) {
      req.reply({
        statusCode: 200,
        body: {
          status: 'active',
        },
      })
    }
  })
}

describe('Configuration tab', () => {
  beforeEach(() => {
    setupIntercepts()
  })

  it('should have a "Configuration" tab', () => {
    cy.visit('/', {
      retryOnStatusCodeFailure: true,
      onLoad: () => {
        cy.get('.MuiTabs-flexContainer')
          .contains('Configuration')
          .should('be.visible')
          .click({ force: true })
      },
    })
  })
})
