/* eslint-disable no-undef */

describe('Instance page should have "Configuration" tab with content', () => {
  beforeEach(() => {
    cy.visit('/')
  })

  it('should have token in local storage', () => {
    cy.window().then((win) => {
      if (!win.localStorage.getItem('token')) {
        window.localStorage.setItem('token', 'demo-token')
      }
    })
  })

  it('should have "Configuration" tab with content', () => {
    cy.once('uncaught:exception', () => false)
    cy.get('.MuiTabs-flexContainer').contains('Configuration')
    cy.get('.MuiBox-root').contains('p').should('have.length.greaterThan', 0)
  })
})
