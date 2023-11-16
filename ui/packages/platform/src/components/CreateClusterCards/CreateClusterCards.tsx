import classNames from 'classnames'
import { StubContainer } from '@postgres.ai/shared/components/StubContainer'
import { icons } from '@postgres.ai/shared/styles/icons'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'
import { ProductCardWrapper } from 'components/ProductCard/ProductCardWrapper'
import { DashboardProps } from 'components/Dashboard/DashboardWrapper'

import Urls from '../../utils/urls'
import { messages } from '../../assets/messages'
import { useStyles } from 'components/CreateDbLabCards/CreateDbLabCards'

export const CreateClusterCards = ({
  isModal,
  props,
  dblabPermitted,
}: {
  isModal?: boolean
  props: DashboardProps
  dblabPermitted: boolean | undefined
}) => {
  const classes = useStyles()

  const createClusterInstanceButton = (provider: string) => {
    props.history.push(Urls.linkClusterInstanceAdd(props, provider))
  }

  const CreateButton = ({ type, title }: { type: string; title: string }) => (
    <ConsoleButtonWrapper
      disabled={!dblabPermitted}
      variant="contained"
      color="primary"
      onClick={() => createClusterInstanceButton(type)}
      title={dblabPermitted ? title : messages.noPermission}
    >
      {type === 'create' ? 'Create' : 'Install'}
    </ConsoleButtonWrapper>
  )

  const productData = [
    {
      title: 'Create Postgres Cluster in your cloud',
      renderDescription: () => (
        <>
          <p>
            Supported cloud platforms include AWS, GCP, Digital Ocean, and
            Hetzner Cloud.
          </p>
          <p>All components are installed within your cloud account.</p>
          <p>Your data remains secure and never leaves your infrastructure.</p>
        </>
      ),
      icon: icons.createDLEIcon,
      actions: [
        {
          id: 'createDblabInstanceButton',
          content: (
            <CreateButton type="create" title="Create Cluster in your cloud" />
          ),
        },
      ],
    },
    {
      title: 'BYOM (Bring Your Own Machines)',
      renderDescription: () => (
        <>
          <p>
            Install on your existing resources, regardless of the machine or
            location. Compatible with both cloud and bare metal infrastructures.
            Your data remains secure and never leaves your infrastructure.
          </p>
          <p>Requirements:</p>
          <ul>
            <li>
              Three or more servers running a supported Linux distro: Ubuntu
              (20.04/22.04), Debian (11/12), CentOS Stream (8/9), Rocky Linux
              (8/9), AlmaLinux (8/9), or Red Hat Enterprise Linux (8/9).
            </li>
            <li>Internet connectivity</li>
          </ul>
        </>
      ),
      icon: icons.installDLEIcon,
      actions: [
        {
          id: 'createDblabInstanceButton',
          content: (
            <CreateButton
              type="install"
              title="Install Cluster on an existing machine"
            />
          ),
        },
      ],
    },
  ]

  return (
    <StubContainer
      className={classNames(
        !isModal && classes.zeroMaxHeight,
        classes.stubContainerProjects,
      )}
    >
      {productData.map((product) => (
        <ProductCardWrapper
          inline
          className={classes.productCardProjects}
          title={product.title}
          actions={product.actions}
          icon={product.icon}
        >
          <div>{product.renderDescription()}</div>
        </ProductCardWrapper>
      ))}
    </StubContainer>
  )
}
