/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import clsx from 'clsx'

import { ClassesType } from '@postgres.ai/platform/src/components/types'

import { ProductCardProps } from 'components/ProductCard/ProductCardWrapper'

interface ProductCardWithStylesProps extends ProductCardProps {
  classes: ClassesType
}

class ProductCard extends Component<ProductCardWithStylesProps> {
  render() {
    const {
      classes,
      children,
      actions,
      inline,
      title,
      icon,
      style,
      className,
    } = this.props

    return (
      <div
        className={clsx(!inline && classes.block, classes.root, className)}
        style={style}
      >
        <div>
          <h1>{title}</h1>
          <div className={classes.contentContainer}>{children}</div>
        </div>

        <div className={classes.bottomContainer}>
          {icon}
          <div className={classes.actionsContainer}>
            {actions?.map((a) => {
              return (
                <span key={a.id} className={classes.buttonSpan}>
                  {a.content}
                </span>
              )
            })}
          </div>
        </div>
      </div>
    )
  }
}

export default ProductCard
