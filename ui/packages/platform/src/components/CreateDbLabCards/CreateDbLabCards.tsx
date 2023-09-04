import { makeStyles } from '@material-ui/core'
import { StubContainer } from '@postgres.ai/shared/components/StubContainer'
import { icons } from '@postgres.ai/shared/styles/icons'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'
import { ProductCardWrapper } from 'components/ProductCard/ProductCardWrapper'
import { DashboardProps } from 'components/Dashboard/DashboardWrapper'

import Urls from '../../utils/urls'
import { messages } from '../../assets/messages'

const useStyles = makeStyles((theme) => ({
  stubContainerProjects: {
    marginRight: '-20px',
    padding: '0 40px',

    '& > div:nth-child(1), & > div:nth-child(2)': {
      minHeight: '350px',
    },

    [theme.breakpoints.down('sm')]: {
      flexDirection: 'column',
    },
  },
  productCardProjects: {
    flex: '1 1 0',
    marginRight: '20px',
    height: 'maxContent',
    gap: 20,
    maxHeight: '100%',
    '& ul': {
      marginBlockStart: '-10px',
      paddingInlineStart: '30px',
    },

    '& svg': {
      width: '206px',
      height: '130px',
    },

    '&:nth-child(1) svg': {
      marginBottom: '-5px',
    },

    '&:nth-child(2) svg': {
      marginBottom: '-20px',
    },

    '& li': {
      listStyleType: 'none',
      position: 'relative',

      '&::before': {
        content: '"-"',
        position: 'absolute',
        left: '-10px',
        top: '0',
      },
    },

    [theme.breakpoints.down('sm')]: {
      flex: '100%',
      marginTop: '20px',
      minHeight: 'auto !important',

      '&:nth-child(1) svg': {
        marginBottom: 0,
      },

      '&:nth-child(2) svg': {
        marginBottom: 0,
      },
    },
  },
}))

export const CreatedDbLabCards = ({
  props,
  dblabPermitted,
}: {
  props: DashboardProps
  dblabPermitted: boolean | undefined
}) => {
  const classes = useStyles()

  const createDblabInstanceButtonHandler = (provider: string) => {
    props.history.push(Urls.linkDbLabInstanceAdd(props, provider))
  }

  const CreateButton = ({ type, title }: { type: string; title: string }) => (
    <ConsoleButtonWrapper
      disabled={!dblabPermitted}
      variant="contained"
      color="primary"
      onClick={() => createDblabInstanceButtonHandler(type)}
      title={dblabPermitted ? title : messages.noPermission}
    >
      {type === 'create' ? 'Create' : 'Install'}
    </ConsoleButtonWrapper>
  )

  const productData = [
    {
      title: 'Create DBLab in your cloud',
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
            <CreateButton type="create" title="Create DBLab in your cloud" />
          ),
        },
      ],
    },
    {
      title: 'BYOM (Bring Your Own Machine)',
      renderDescription: () => (
        <>
          <p>
            Install on your existing resources, regardless of the machine or
            location. Compatible with both cloud and bare metal infrastructures.
            Your data remains secure and never leaves your infrastructure.
          </p>
          <p>Requirements:</p>
          <ul>
            <li>Ubuntu 20.04 or newer</li>
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
              title="Install DLE on an existing machine"
              title="Install DBLab on an existing machine"
            />
          ),
        },
      ],
    },
  ]

  return (
    <StubContainer className={classes.stubContainerProjects}>
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
