/* eslint-disable no-undef */

describe('Configuration tab', () => {
  before(() => {
    // Set the token in local storage
    cy.window().then((win) => {
      if (!win.localStorage.getItem('token')) {
        win.localStorage.setItem('token', 'demo-token')
      }
    })
  })

  it('should have token in local storage', () => {
    // Check if the token exists in local storage
    cy.window()
      .should('have.property', 'localStorage')
      .and('have.property', 'token', 'demo-token')
  })

  it('should have "Configuration" tab with form inputs', () => {
    // Visit the page
    cy.visit('/', {
      retryOnStatusCodeFailure: true,
      onLoad: () => {
        // Click on the "Configuration" tab
        cy.get('.MuiTabs-flexContainer').contains('Configuration').click({
          force: true,
        })

        // Check for elements on the "Configuration" tab
        cy.get('input[type="text"]').should('exist')
        cy.get('input[type="checkbox"]').should('exist')
        cy.get('button[type="button"]').should('exist')

        // Click on the "Cancel" button within the "Configuration" tab
        cy.get('button').contains('Cancel').click({
          force: true,
        })

        // Check if the URL has changed to "/instance"
        cy.url().should('eq', Cypress.config().baseUrl + '/instance')
      },
    })
  })
})
