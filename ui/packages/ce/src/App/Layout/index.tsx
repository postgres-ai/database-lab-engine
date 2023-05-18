import styles from './styles.module.scss'
import { StickyTopBar } from 'App/Menu/StickyTopBar'

type Props = {
  displayStickyBanner: boolean | undefined
  menu: React.ReactNode
  children: React.ReactNode
}

export const Layout = (props: Props) => {
  return (
    <div className={styles.root}>
      {props.displayStickyBanner && <StickyTopBar />}
      <div className={styles.menu}>{props.menu}</div>
      <div id="content-container" className={styles.content}>
        {props.children}
      </div>
    </div>
  )
}
