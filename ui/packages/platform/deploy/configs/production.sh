#--------------------------------------------------------------------------
# Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
# All Rights Reserved
# Unauthorized copying of this file, via any medium is strictly prohibited
# Proprietary and confidential
#--------------------------------------------------------------------------

export PORT=3000
export REPLICAS=1
export REACT_APP_API_SERVER="https://postgres.ai/api/general"
export REACT_APP_WS_SERVER="wss://postgres.ai/websockets"
export REACT_APP_SIGNIN_URL="http://console.postgres.ai/signin"
export REACT_APP_EXPLAIN_DEPESZ_SERVER="https://explain-depesz.postgres.ai/"
export REACT_APP_EXPLAIN_PEV2_SERVER="https://postgres.ai/explain-pev2"
export REACT_APP_AUTH_URL="https://postgres.ai/api/auth"
export REACT_APP_ROOT_URL="https://postgres.ai"
export PUBLIC_URL=""

# Public Stripe key, it is ok to keep it here.
export REACT_APP_STRIPE_API_KEY="xxx"

# Sentry.
export REACT_APP_SENTRY_DSN="https://91517477289e477cb8880f2f07a82632@sentry.postgres.ai/2"

# AI Bot
export REACT_APP_WS_URL="wss://postgres.ai/ai-bot-ws/" # don't forget trailing slash!