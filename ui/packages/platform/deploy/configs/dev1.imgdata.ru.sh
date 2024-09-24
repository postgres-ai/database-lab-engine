#--------------------------------------------------------------------------
# Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
# All Rights Reserved
# Unauthorized copying of this file, via any medium is strictly prohibited
# Proprietary and confidential
#--------------------------------------------------------------------------

export PORT=3000
export REPLICAS=1
export REACT_APP_API_SERVER="http://dev1.imgdata.ru:9508"
export REACT_APP_WS_SERVER="ws://dev1.imgdata.ru:9509"
export REACT_APP_SIGNIN_URL="http://dev1.imgdata.ru:5000"
export REACT_APP_EXPLAIN_DEPESZ_SERVER="https://explain-depesz.postgres.ai/"
export REACT_APP_EXPLAIN_PEV2_SERVER="https://postgres.ai/explain-pev2/"
export REACT_APP_AUTH_URL="http://localhost:3001"
export REACT_APP_ROOT_URL="https://postgres.ai"
export PUBLIC_URL=""

# Public Stripe key, it is ok to keep it here.
export REACT_APP_STRIPE_API_KEY="xxx"
export REACT_APP_SENTRY_DSN=""
