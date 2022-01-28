import cn from 'classnames'
import { NavLink } from 'react-router-dom'

import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'

import styles from './styles.module.scss'

type BaseProps = {
  children: React.ReactNode
  className?: string
  icon?: React.ReactNode
  isCollapsed?: boolean
}

type ButtonProps = BaseProps & {
  type?: 'button' | 'submit'
  onClick?: React.MouseEventHandler<HTMLButtonElement>
}

type LinkProps = BaseProps & {
  type: 'link'
  to: string
  activeClassName: string
}

type GatewayLinkProps = BaseProps & {
  type: 'gateway-link'
  href: string
}

type Props = ButtonProps | LinkProps | GatewayLinkProps

export const Button = (props: Props) => {
  const className = cn(
    styles.root,
    props.isCollapsed && styles.collapsed,
    props.className,
  )

  const children = (
    <>
      {props.icon && (
        <span
          className={cn(styles.icon, props.isCollapsed && styles.collapsed)}
        >
          {props.icon}
        </span>
      )}
      {!props.isCollapsed && props.children}
    </>
  )

  if (!props.type || props.type === 'button' || props.type === 'submit')
    return (
      <button className={className} onClick={props.onClick}>
        {children}
      </button>
    )

  if (props.type === 'link')
    return (
      <NavLink
        className={className}
        to={props.to}
        activeClassName={props.activeClassName}
      >
        {children}
      </NavLink>
    )

  if (props.type === 'gateway-link')
    return (
      <GatewayLink className={className} href={props.href}>
        {children}
      </GatewayLink>
    )

  return null
}
