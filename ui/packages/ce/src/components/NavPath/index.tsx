import React from 'react'
import { NavLink } from 'react-router-dom'
import cn from 'classnames'

import styles from './styles.module.scss'

export type NavRoute = {
  name: string
  path: string
}

type Props = {
  routes: NavRoute[]
  className?: string
}

export const NavPath = (props: Props) => {
  return (
    <nav className={cn(styles.root, props.className)}>
      {props.routes.map((route, i) => {
        const isLast = (i + 1) === props.routes.length
        const nameWithIndent = route.name.replace('/', ' / ')

        return (
          <React.Fragment key={i}>
            <NavLink
              exact
              to={route.path}
              className={styles.link}
              activeClassName={styles.active}
            >
              {nameWithIndent}
            </NavLink>
            { !isLast && <span className={styles.divider}>/</span> }
          </React.Fragment>
        )
      })}
    </nav>
  )
}
