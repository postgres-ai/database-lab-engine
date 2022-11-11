/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import settings from './settings'

export interface DataType {
  port: number | null
  pgPort: number | null | string
  sshPort: number | null | string
  statementTimeout: number | null | string
  ssDatabaseName: string
  pgSocketDir: string
  psqlBinary: string
  sshKeysPath: string
  collectPeriod: number | string
  hosts: string
  projectName: string
  databaseName: string
  databaseUserName: string
  apiToken: string
  password: string
  connectionType: string
}

const cfgGen = {
  requiredParamsFilled: function (data: DataType) {
    if (
      data.hosts.trim() === '' ||
      data.projectName === '' ||
      data.databaseName === '' ||
      data.databaseUserName === '' ||
      data.apiToken === '' ||
      data.password === '' ||
      data.connectionType === ''
    ) {
      return false
    }
    return true
  },

  uniqueHosts: function (hosts: string) {
    const hostsArray = hosts.split(/[;,(\s)(\n)(\r)(\t)(\r\n)]/)
    const newHostsArray = []

    for (const i in hostsArray) {
      if (hostsArray[i] !== '' && newHostsArray.indexOf(hostsArray[i]) === -1) {
        newHostsArray.push(hostsArray[i])
      }
    }

    return newHostsArray.join(';')
  },

  generateFromSourcesInstruction: function (data: {
    connectionType: string
    hosts: string
    projectName: string
    password: string
    databaseUserName: string
    collectPeriod: number | string
    databaseName: string
    apiToken: string
  }) {
    let hostsType = 'CHECKUP_HOSTS'

    if (!this.requiredParamsFilled(data)) {
      return '# Not enough data'
    }

    if (data.connectionType === 'ssh') {
      hostsType = 'SSH_CHECKUP_HOSTS'
    }

    if (data.connectionType === 'pg') {
      hostsType = 'PG_CHECKUP_HOSTS'
    }

    const hosts = this.uniqueHosts(data.hosts).split(';').join(' ')
    let result = `# Create config file
cat <<EOF > ${data.projectName}.yml
${this.generateCheckupConfig(data)}
EOF

# Start check health of Postgres databases
`

    if (data.password === 'inputpassword') {
      result =
        result +
        `echo -e "\\n\\nEnter the password for DB user "${data.databaseUserName}": " \\
&& read -s -p "" DB_PWD \\
&& PGPASSWORD="$\{DB_PWD}" \\
`
    }

    result =
      result +
      `CHECKUP_CONFIG_PATH=./${data.projectName}.yml \\
  CHECKUP_SNAPSHOT_DISTANCE_SECONDS=${data.collectPeriod} \\
  ${hostsType}="${hosts}" \\
  ./run_checkup.sh
`

    return result
  },

  generateDockerInstruction: function (data: DataType) {
    let hostsType = 'CHECKUP_HOSTS'

    if (!this.requiredParamsFilled(data)) {
      return '# Not enough data'
    }

    if (data.connectionType === 'ssh') {
      hostsType = 'SSH_CHECKUP_HOSTS'
    }

    if (data.connectionType === 'pg') {
      hostsType = 'PG_CHECKUP_HOSTS'
    }

    const hosts = this.uniqueHosts(data.hosts).split(';').join(' ')
    let result = `# Create config file
cat <<EOF > ${data.projectName}.yml
${this.generateCheckupConfig(data)}
EOF

# Start check health of Postgres databases
`

    if (data.password === 'inputpassword') {
      result =
        result +
        `echo -e "\\n\\nEnter the password for DB user "${data.databaseUserName}": " \\
&& read -s -p "" DB_PWD \\
&& PGPASSWORD="$\{DB_PWD} \\
&& `
    }

    result =
      result +
      `docker run \\
  -v $(pwd)/${data.projectName}.yml:/${data.projectName}.yml \\
  -v $(pwd)/artifacts:/artifacts \\`

    if (data.sshKeysPath !== '') {
      result =
        result +
        `
  -v ${data.sshKeysPath}:/root/.ssh \\`
    }

    result =
      result +
      `
  -e CHECKUP_CONFIG_PATH="./${data.projectName}.yml" \\
  -e ${hostsType}="${hosts}" \\
  -e CHECKUP_SNAPSHOT_DISTANCE_SECONDS=${data.collectPeriod} \\
  -e PGPASSWORD="$\{DB_PWD}" \\
  registry.gitlab.com/postgres-ai/postgres-checkup:latest \\
  bash run_checkup.sh
`

    return result
  },

  generateCheckupConfig: function (data: DataType) {
    let result = `- project: ${data.projectName}
  dbname: ${data.databaseName}
  username: ${data.databaseUserName}
  epoch: 1
  upload-api-url: ${settings.apiServer}`

    if (data.apiToken && data.apiToken !== '') {
      result =
        result +
        `
  upload-api-token: ${data.apiToken}`
    }

    if (data.pgPort && data.pgPort !== 5432) {
      result =
        result +
        `
  pg-port ${data.pgPort}`
    }

    if (data.sshPort && data.sshPort !== 22) {
      result =
        result +
        `
  ssh-port ${data.sshPort}`
    }

    if (data.statementTimeout && data.statementTimeout !== 30) {
      result =
        result +
        `
  statement-timeout ${data.statementTimeout}`
    }

    if (data.ssDatabaseName && data.ssDatabaseName !== '') {
      result =
        result +
        `
  ss-dbname: ${data.ssDatabaseName}`
    }

    if (data.pgSocketDir && data.pgSocketDir !== '') {
      result =
        result +
        `
  pg-socket-dir: ${data.pgSocketDir}`
    }

    if (data.psqlBinary && data.psqlBinary !== '') {
      result =
        result +
        `
  psql-binary: ${data.psqlBinary}`
    }

    return result
  },

  generateRunCheckupSh: function (data: DataType) {
    let result = `#!/bin/bash
#run_checkup.sh

`
    const hosts = this.uniqueHosts(data.hosts).split(';').join(' ')

    // Collect command
    let collectCmd = `  ./checkup collect \\
    --hostname "$\{host}" \\
    --project "${data.projectName}" \\
    -U "${data.databaseUserName}" \\
    -e "1" \\
    -d ${data.databaseName}`

    if (data.port && data.port !== 5432) {
      collectCmd =
        collectCmd +
        ` \\
    --port ${data.port}`
    }

    if (data.statementTimeout && data.statementTimeout !== 30) {
      collectCmd =
        collectCmd +
        ` \\
    -S ${data.statementTimeout}`
    }

    if (data.ssDatabaseName && data.ssDatabaseName !== '') {
      collectCmd =
        collectCmd +
        ` \\
    --ss-dbname ${data.ssDatabaseName}`
    }

    if (data.pgSocketDir && data.pgSocketDir !== '') {
      collectCmd =
        collectCmd +
        ` \\
    --pg-socket-dir "${data.pgSocketDir}"`
    }

    if (data.psqlBinary && data.psqlBinary !== '') {
      collectCmd =
        collectCmd +
        ` \\
    --psql-binary "${data.psqlBinary}"`
    }

    result =
      result +
      `echo "Collect the first set of snapshots (only for K*** reports)..."
for host in ${hosts} ; do
  ${collectCmd} \\
  --file resources/checks/K000_query_analysis.sh
done

echo "The first set of snapshots has been created. Wait ${data.collectPeriod} seconds..."
sleep "${data.collectPeriod}"
# the distance ^^^ should be large enough to get good data, at least 10 minutes

echo "Collect the second set of snapshots and build reports..."
for host in ${hosts} ; do
  ${collectCmd}
done

`

    result =
      result +
      `echo "Generate human readable reports..."
./checkup process --project ${data.projectName} --html --pdf

`

    if (data.apiToken && data.apiToken !== '') {
      result =
        result +
        `echo "Upload report files to platform..."
./checkup upload --project ${data.projectName} --upload-api-token "${data.apiToken}" --upload-api-url "${settings.apiServer}"
`
    } else {
      result =
        result +
        `echo "Upload report files to platform..."
./checkup upload --project ${data.projectName} --upload-api-token "%TOKEN%" --upload-api-url "${settings.apiServer}"
`
    }
    /* eslint-enable max-len */

    return result
  },
}

export default cfgGen
