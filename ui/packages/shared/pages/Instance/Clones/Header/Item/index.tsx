import styles from './styles.module.scss'

type Props = {
  value: React.ReactNode
  children: React.ReactNode
}

export const Item = (props: Props) => {
  return (
    <div className={styles.root}>
      <div className={styles.value}>{props.value}</div>
      <div className={styles.description}>{props.children}</div>
    </div>
  )
}
