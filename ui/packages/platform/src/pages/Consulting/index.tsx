import React, { useEffect } from "react";
import ConsolePageTitle from "../../components/ConsolePageTitle";
import Alert from "@mui/material/Alert";
import { Grid, Typography } from "@mui/material";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box/Box";
import { observer } from "mobx-react-lite";
import { consultingStore } from "../../stores/consulting";
import { ConsultingWrapperProps } from "./ConsultingWrapper";
import { makeStyles } from "@material-ui/core";
import { PageSpinner } from "@postgres.ai/shared/components/PageSpinner";
import { ProductCardWrapper } from "../../components/ProductCard/ProductCardWrapper";
import { Link } from "@postgres.ai/shared/components/Link2";
import { ConsoleBreadcrumbsWrapper } from "../../components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper";
import { formatPostgresInterval } from "./utils";
import { TransactionsTable } from "./TransactionsTable/TransactionsTable";



const useStyles = makeStyles((theme) => ({
  sectionLabel: {
    fontSize: '14px!important',
    fontWeight: '700!important' as 'bold',
  },
  productCardProjects: {
    flex: '1 1 0',
    marginRight: '20px',
    height: 'maxContent',
    gap: 20,
    maxHeight: '100%',

    '& svg': {
      width: '206px',
      height: '130px',
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

export const Consulting = observer((props: ConsultingWrapperProps) => {
  const { orgId, orgData, match } = props;

  const classes = useStyles();

  useEffect(() => {
    if (orgId) {
      consultingStore.getOrgBalance(orgId);
      consultingStore.getTransactions(orgId);
    }
  }, [orgId]);

  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      org={match.params.org}
      breadcrumbs={[{ name: "Consulting" }]}
    />
  )

  if (consultingStore.loading) {
    return (
      <Box>
        {breadcrumbs}
        <ConsolePageTitle title={"Consulting"} />
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
          <PageSpinner />
        </Box>
      </Box>
    )
  }

  if (orgData.consulting_type === null) {
    return (
      <Box>
        {breadcrumbs}
        <ConsolePageTitle title={"Consulting"} />
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
          <ProductCardWrapper
            inline
            className={classes.productCardProjects}
            title="Not a customer yet"
            actions={[
              {
                id: 'learn-more',
                content: (<Link to="https://postgres.ai/consulting" external target="_blank">Learn more</Link>)
              }
            ]}
          >
            <p>
              Your organization is not a consulting customer yet. To learn more about Postgres.AI consulting, visit this page: <Link to="https://postgres.ai/consulting" external target="_blank">Consulting</Link>.
            </p>
            <p>
              Reach out to the team to discuss consulting opportunities: <Link to="mailto:consulting@postgres.ai" external target="_blank">consulting@postgres.ai</Link>.
            </p>
          </ProductCardWrapper>
          </Box>
      </Box>
    )
  }

  return (
    <div>
      {breadcrumbs}
      <ConsolePageTitle title={"Consulting"} />
      <Grid container spacing={3}>
        <Grid item xs={12} md={8}>
          <Box>
            <Typography variant="body1" sx={{ fontSize: 14 }}>
              Thank you for choosing Postgres.AI as your PostgreSQL consulting partner. Your plan: {orgData.consulting_type.toUpperCase()}.
            </Typography>
          </Box>
        </Grid>
        <Grid item xs={12} md={8}>
          <Box>
            <Typography variant="h6" classes={{root: classes.sectionLabel}}>
              Issue tracker (GitLab):
            </Typography>
            <Typography variant="body1" sx={{ marginTop: 1, fontSize: 14}}>
              <Link to={`https://gitlab.com/postgres-ai/postgresql-consulting/support/${orgData.alias}`} external target="_blank">
                https://gitlab.com/postgres-ai/postgresql-consulting/support/{orgData.alias}
              </Link>
            </Typography>
          </Box>
        </Grid>
        <Grid item xs={12} md={8}>
          <Box>
            <Typography variant="h6" classes={{root: classes.sectionLabel}}>
              Book a Zoom call:
            </Typography>
            <Typography variant="body1" sx={{ marginTop: 1, fontSize: 14}}>
              <Link to={`https://calend.ly/postgres`} external target="_blank">
                https://calend.ly/postgres
              </Link>
            </Typography>
          </Box>
        </Grid>
        <Grid item xs={12} md={8}>
          <Box>
            <Typography variant="h6" classes={{root: classes.sectionLabel}}>
              Email:
            </Typography>
            <Typography variant="body1" sx={{ marginTop: 1, fontSize: 14}}>
              <Link to={`mailto:consulting@postgres.ai`} external target="_blank">
                consulting@postgres.ai
              </Link>
            </Typography>
          </Box>
        </Grid>
        {consultingStore.orgBalance?.[0]?.balance?.charAt(0) === '-' && <Grid item xs={12} md={8}>
          <Alert severity="warning">Consulting hours overdrawn</Alert>
        </Grid>}
        {orgData.consulting_type === 'retainer' && <Grid item xs={12} md={8}>
          <Typography variant="h6" classes={{root: classes.sectionLabel}}>
            Retainer balance:
          </Typography>
          <Typography variant="h5" sx={{ marginTop: 1}}>
            {formatPostgresInterval(consultingStore.orgBalance?.[0]?.balance || '00') || 0}
          </Typography>
        </Grid>}
        {orgData.consulting_type === 'retainer' && <Grid item xs={12} md={8}>
          <Box>
            <Button variant="contained" component="a" href="https://buy.stripe.com/7sI5odeXt3tB0Eg3cm" target="_blank">
              Replenish consulting hours
            </Button>
          </Box>
        </Grid>}

        {orgData.consulting_type === 'retainer' && <Grid item xs={12} md={8}>
          <Typography variant="h6" classes={{root: classes.sectionLabel}}>
            Activity:
          </Typography>
          {
            consultingStore.transactions?.length === 0
              ? <Typography variant="body1" sx={{ marginTop: 1}}>
                  No activity yet
                </Typography>
              : <TransactionsTable transactions={consultingStore.transactions} alias={orgData.alias} />
          }
        </Grid>}
      </Grid>
    </div>
  );
});

