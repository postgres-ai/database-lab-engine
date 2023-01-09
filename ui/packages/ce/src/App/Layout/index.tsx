import styles from './styles.module.scss'

type Props = {
  menu: React.ReactNode
  children: React.ReactNode
}

export const Layout = (props: Props) => {
  return (
    <div className={styles.root}>
      <div className={styles.menu}>{props.menu}</div>
      <div id="content-container" className={styles.content}>{props.children}</div>
    </div>
  )
}
