import { ClassesType } from 'components/types'

export const SetupStep = ({ classes }: { classes: ClassesType }) => (
  <>
    <p className={classes.title}>1. Set up your machine</p>
    <ul className={classes.ul}>
      <li>
        Obtain a machine running Ubuntu 22.04 (although other versions may work,
        we recommend using an LTS version for optimal compatibility).
      </li>
      <li>
        Attach an empty disk that is at least twice the size of the database you
        plan to use with DBLab.
      </li>
      <li>
        Ensure that your SSH public key is added to the machine (in
        <code className={classes.code}>~/.ssh/authorized_keys</code>), allowing
        for secure SSH access.
      </li>
    </ul>
  </>
)
