/*
2019 © Postgres.ai
*/

package provision

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"../ec2ctrl"
	"../log"

	"github.com/docker/machine/libmachine/ssh"
	"github.com/tkanos/gonfig"
)

var awsValidDurations = []int64{60, 120, 180, 240, 300, 360}

const (
	DOCKER_NAME      = "inside-instance-docker"
	PRICE_MULTIPLIER = 1.1
	PG_PROCESS_CHECK = "ps ax | grep postgres | grep -v \"grep\" | awk '{print $5}' | grep \"postgres:\" 2>/dev/null || echo ''"
)

type AwsConfig struct {
	Ec2           ec2ctrl.Ec2Configuration `yaml:"ec2"`
	EbsVolumeId   string                   `yaml:"ebsVolumeId"`
	SshTunnelPort uint                     `yaml:"sshTunnelPort"`
}

type provisionAws struct {
	provision
	ec2ctrl           ec2ctrl.Ec2Ctrl
	instanceId        string
	instanceIp        string
	sessionId         string
	sshClient         ssh.Client
	dockerContainerId string
}

func NewProvisionAws(config Config) (Provision, error) {
	ec2ctrl := ec2ctrl.NewEc2Ctrl(config.Aws.Ec2)

	provisionAws := &provisionAws{
		ec2ctrl:    *ec2ctrl,
		instanceId: "",
		instanceIp: "",
	}

	provisionAws.config = config

	return provisionAws, nil
}

func isValidConfigModeAws(config Config) bool {
	result := true

	if config.Aws.Ec2.AwsInstanceType == "" {
		log.Err("AwsInstanceType cannot be empty.")
		result = false
	}

	if ec2ctrl.RegionDetails[config.Aws.Ec2.AwsRegion] == nil {
		log.Err("Wrong configuration AwsRegion value.")
		result = false
	}

	if len(config.Aws.Ec2.AwsZone) != 1 {
		log.Err("Wrong configuration AwsZone value (must be exactly 1 letter).")
		result = false
	}

	duration := config.Aws.Ec2.AwsBlockDurationMinutes
	isValidDuration := false
	for _, validDuration := range awsValidDurations {
		if duration == validDuration {
			isValidDuration = true
			break
		}
	}
	if !isValidDuration {
		log.Err("Wrong configuration AwsBlockDurationMinutes value.")
		result = false
	}

	if config.Aws.Ec2.AwsKeyName == "" {
		log.Err("AwsKeyName cannot be empty.")
		result = false
	}

	if config.Aws.Ec2.AwsKeyPath == "" {
		log.Err("AwsKeyPath cannot be empty.")
		result = false
	}

	if _, err := os.Stat(config.Aws.Ec2.AwsKeyPath); err != nil {
		log.Err("Wrong configuration AwsKeyPath value. File does not exits.")
		result = false
	}

	if config.InitialSnapshot == "" {
		log.Err("InitialSnapshot cannot be empty.")
		result = false
	}

	return result
}

func (j *provisionAws) NewAwsSession() *Session {
	session := &Session{
		Id:       j.sessionId,
		Host:     "localhost",
		Port:     j.config.Aws.SshTunnelPort,
		User:     j.config.DbUsername,
		Password: j.config.DbPassword,
	}

	return session
}

// Provision interface implementaion.
func (j *provisionAws) Init() error {
	err := j.readState()
	if err != nil {
		log.Err(err)
	}

	err = j.terminate()
	if err != nil {
		return err
	}

	_, err = j.StartSession()
	if err != nil {
		return err
	}

	return nil
}

func (j *provisionAws) Reinit() error {
	return j.Init()
}

func (j *provisionAws) StartSession(options ...string) (*Session, error) {
	snapshot := j.config.InitialSnapshot
	if len(options) > 0 && len(options[0]) > 0 {
		snapshot = options[0]
	}

	if j.sessionId != "" {
		log.Dbg("Session has been started already.")

		return j.NewAwsSession(), nil
	}
	j.sessionId = strconv.FormatInt(time.Now().UnixNano(), 10)

	// Check instance existance.
	if j.instanceId != "" {
		out, err := j.dockerRunCommand("echo 1")
		out = strings.Trim(out, "\n")
		if err != nil || out != "1" {
			j.stopInstance()
		}
	}

	if j.instanceId == "" {
		err := j.startWorkingInstance()
		if err != nil {
			return nil, fmt.Errorf("Cannot start the working instance. %v", err)
		}
	}

	err := j.dockerRollbackZfsSnapshot(snapshot)
	if err != nil {
		return nil, fmt.Errorf("Cannot rollback the state of the database. %v", err)
	}

	err = j.dockerCreateDbUser()
	if err != nil {
		return nil, fmt.Errorf("Cannot create a database user. %v", err)
	}

	err = j.writeState()
	if err != nil {
		return nil, fmt.Errorf("Cannot save the state. %v", err)
	}

	err = j.createSshTunnel()
	if err != nil {
		return nil, err
	}

	return j.NewAwsSession(), nil
}

func (j *provisionAws) StopSession(session *Session) error {
	return nil
}

func (j *provisionAws) ResetSession(session *Session, options ...string) error {
	snapshot := j.config.InitialSnapshot
	if len(options) > 0 {
		snapshot = options[0]
	}

	err := j.dockerRollbackZfsSnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("Cannot reset database state. %v", err)
	}

	err = j.dockerCreateDbUser()
	if err != nil {
		return fmt.Errorf("Cannot interact with database after state reset. %v", err)
	}

	return nil
}

func (j *provisionAws) CreateSnapshot(name string) error {
	return j.createZfsSnapshot(name)
}

func (j *provisionAws) RunPsql(session *Session, command string) (string, error) {
	// TODO(anatoly): Implement.
	return "", fmt.Errorf("Unsupported in `aws` mode.")
}

// Private methods.
func (j *provisionAws) terminate() error {
	CloseSshTunnel(j.instanceIp)
	j.sessionId = ""
	return j.writeState()
}

func (j *provisionAws) getStateFilePath() string {
	bindir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir, _ := filepath.Abs(filepath.Dir(bindir))
	return dir + string(os.PathSeparator) + STATE_DIR + string(os.PathSeparator) + STATE_FILE
}

// Read state informatation from file.
func (j *provisionAws) readState() error {
	log.Dbg("Reading AWS provision state...")

	state := State{}
	err := gonfig.GetConf(j.getStateFilePath(), &state)
	if err != nil {
		log.Err("ReadState: Cannot read state file.", err)
		return fmt.Errorf("Cannot read state file. %v", err)
	}

	if state.InstanceId == j.instanceId {
		log.Dbg("ReadState: saved instance id is equal to the current instance id", state.InstanceId, j.instanceId)
		return nil
	}

	if j.instanceId != "" {
		return fmt.Errorf("State read, but current instance id differs from the read instance id.")
	}

	isRunning, err := j.ec2ctrl.IsInstanceRunning(state.InstanceId)
	if err != nil {
		return err
	}
	if !isRunning { // TODO(anatoly): another excess res?
		log.Dbg("ReadState: instance not running.")
		return nil
	}

	j.instanceId = state.InstanceId
	j.instanceIp, err = j.ec2ctrl.GetPublicInstanceIpAddress(state.InstanceId)
	err = j.startInstanceSsh()
	if err != nil {
		j.stopInstance()
		return err
	}

	j.dockerContainerId = state.DockerContainerId
	out, derr := j.dockerRunCommand("echo 1")
	out = strings.Trim(out, "\n")
	if out != "1" || derr != nil {
		j.stopInstance()
		return fmt.Errorf("Cannot connect to Docker. %s %v", out, derr)
	}

	j.sessionId = state.SessionId
	log.Dbg("ReadState:", log.OK)

	return nil
}

// Write state information to file.
func (j *provisionAws) writeState() error {
	log.Dbg("Writing AWS provision state...")

	state := State{
		InstanceId:        j.instanceId,
		InstanceIp:        j.instanceIp,
		DockerContainerId: j.dockerContainerId,
		SessionId:         j.sessionId,
	}

	f, err := os.Create(j.getStateFilePath())
	if err != nil {
		return err
	}

	b, err := json.Marshal(state)
	if err != nil {
		return err
	}

	log.Dbg("AWS provision state:", string(b))

	wrote, err := f.Write(b)
	if err != nil || wrote <= 0 {
		return err
	}
	f.Close()

	log.Dbg("WriteState:", log.OK)

	return nil
}

// Start new EC2 instance.
func (j *provisionAws) startInstance() error {
	log.Msg("Starting EC2 instance...")

	price := j.ec2ctrl.GetHistoryInstancePrice()
	price = price * PRICE_MULTIPLIER

	j.instanceId = ""

	var err error

	j.instanceId, err = j.ec2ctrl.CreateSpotInstance(price)
	if err != nil {
		log.Err(err)
		return err
	}

	j.instanceIp, err = j.ec2ctrl.GetPublicInstanceIpAddress(j.instanceId)
	if err != nil {
		log.Err(err)
		return fmt.Errorf("Unable to get the IP of the instance. Check that the instance has started %v.", err)
	}

	log.Msg("The EC2 instance is ready. Instance id is " + log.YELLOW + j.instanceId + log.END)
	log.Msg("To connect to the instance use: " + log.WHITE +
		"ssh -o 'StrictHostKeyChecking no' -i " + j.config.Aws.Ec2.AwsKeyPath +
		" ubuntu@" + j.instanceIp + log.END)

	return nil
}

func (j *provisionAws) stopInstance() error {
	if len(j.instanceId) == 0 {
		return nil
	}

	_, err := j.ec2ctrl.TerminateInstance(j.instanceId)

	j.instanceId = ""
	j.instanceIp = ""
	j.dockerContainerId = ""

	return err
}

// Start SSH accecss to instance.
func (j *provisionAws) startInstanceSsh() error {
	log.Dbg("Establishing connection to the instance using SSH...")

	var err error

	j.sshClient, err = j.ec2ctrl.GetInstanceSshClient(j.instanceId)
	if err != nil || j.sshClient == nil {
		return fmt.Errorf("Cannot connect to the instance using SSH. %v", err)
	}

	return j.ec2ctrl.WaitInstanceForSsh()
}

// Attach EC2 drive to instance which ZFS formatted and has database snapshot.
// TODO(anatoly): Rename pancake.
func (j *provisionAws) attachZfsPancake() error {
	log.Msg("Attaching pancake drive...")

	_, err := j.ec2ctrl.RunInstanceSshCommand("sudo apt-get update", j.config.Debug)
	if err != nil {
		return err
	}
	_, err = j.ec2ctrl.RunInstanceSshCommand("sudo apt-get install -y zfsutils-linux", j.config.Debug)
	if err != nil {
		return err
	}
	_, err = j.ec2ctrl.RunInstanceSshCommand("sudo sh -c \"mkdir /home/storage\"", j.config.Debug)
	if err != nil {
		return err
	}
	_, err = j.ec2ctrl.AttachInstanceVolume(j.instanceId, j.config.Aws.EbsVolumeId, "/dev/xvdc")
	if err != nil {
		return fmt.Errorf("Cannot attach the persistent disk to the instance, %v.", err)
	}
	_, err = j.ec2ctrl.RunInstanceSshCommand("sudo zpool import -R / zpool", j.config.Debug)
	if err != nil {
		return err
	}
	_, err = j.ec2ctrl.RunInstanceSshCommand("sudo df -h /home/storage", j.config.Debug)
	if err != nil {
		return err
	}

	out, err := j.ec2ctrl.RunInstanceSshCommand("grep MemTotal /proc/meminfo | awk '{print $2}'", j.config.Debug)
	if err != nil {
		return err
	}
	out = strings.Trim(out, "\n")
	memTotalKb, _ := strconv.Atoi(out)
	arcSizeB := memTotalKb / 100 * 30 * 1024
	if arcSizeB < 1073741824 {
		arcSizeB = 1073741824 // 1 GiB
	}

	size := strconv.FormatInt(int64(arcSizeB), 10)
	command := fmt.Sprintf("echo %s | sudo tee /sys/module/zfs/parameters/zfs_arc_max", size)
	_, err = j.ec2ctrl.RunInstanceSshCommand(command, j.config.Debug)
	if err != nil {
		return err
	}

	return nil
}

// Start docker inside instance.
func (j *provisionAws) startDocker() error {
	log.Msg("Installing docker...")

	_, err := j.ec2ctrl.RunInstanceSshCommand("sudo apt install -y docker.io", j.config.Debug)
	if err != nil {
		return err
	}

	out, err := j.ec2ctrl.RunInstanceSshCommand("docker --version", false)
	if err != nil {
		return err
	}
	log.Msg("Installed docker version: " + log.WHITE + strings.Trim(out, "\n") + log.END)

	log.Msg("Pulling docker image...")
	_, err = j.ec2ctrl.RunInstanceSshCommand(
		"sudo docker pull \"postgresmen/postgres-nancy:"+j.config.PgVersion+"\" 2>&1 | "+
			"grep -e 'Pulling from' -e Digest -e Status -e Error", j.config.Debug)
	if err != nil {
		j.stopInstance()
		return fmt.Errorf("Cannot pull docker image, %v", err)
	}

	log.Msg("Starting docker...")
	out, err = j.ec2ctrl.RunInstanceSshCommand("sudo docker run --cap-add SYS_ADMIN "+
		"--name=\""+DOCKER_NAME+"\" -p 5432:5432 -v /home/ubuntu:/machine_home "+
		"-v /home/storage:/storage "+
		"-dit \"postgresmen/postgres-nancy:"+j.config.PgVersion+"\"", j.config.Debug)
	if err != nil {
		j.stopInstance()
		return fmt.Errorf("Cannot start Docker, %v.", err)
	}
	j.dockerContainerId = strings.Trim(out, "\n")

	log.Msg("Docker container hash is  " + log.YELLOW + j.dockerContainerId + log.END)
	log.Msg("To connect to Docker use: " + log.WHITE + "sudo docker exec -it " + DOCKER_NAME + " bash" + log.END)
	log.Msg("or:                       " + log.WHITE + "sudo docker exec -i " + j.dockerContainerId + " bash" + log.END)

	return nil
}

// Execute bash command inside docker container.
func (j *provisionAws) dockerRunCommand(command string, options ...bool) (string, error) {
	debug := j.config.Debug
	if len(options) > 0 {
		debug = options[0]
	}

	command = strings.ReplaceAll(command, "\"", "\\\"")
	command = strings.ReplaceAll(command, "\n", " ") // For multiline SQL code.

	cId := j.dockerContainerId
	cmd := fmt.Sprintf(`sudo docker exec -i %s bash -c "%s"`, cId, command)

	return j.ec2ctrl.RunInstanceSshCommand(cmd, debug)
}

// Start Postgres inside Docker.
func (j *provisionAws) dockerStartPostgres() error {
	log.Dbg("Starting Postgres...")
	var out string
	var err error

	cnt := 0
	for true {
		out, err = j.dockerRunCommand(PG_PROCESS_CHECK)
		out = strings.Trim(out, "\n ")
		if out != "" && err == nil {
			log.Dbg("Postgres has been started.")
			return nil
		}

		cnt++
		if cnt > 900 { // 15 minutes = 900 seconds
			return fmt.Errorf("Postgres could not be started in 15 minutes.")
		}

		_, err = j.dockerRunCommand("sudo pg_ctlcluster " + j.config.PgVersion + " main start || true")

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("Cannot start Postgres. Unreachable code.")
}

// Stop Postgres inside Docker container.
func (j *provisionAws) dockerStopPostgres() error {
	log.Dbg("Stopping Postgres...")
	var out string
	var err error

	cnt := 0
	for true {
		out, err = j.dockerRunCommand(PG_PROCESS_CHECK)
		out = strings.Trim(out, "\n ")
		if out == "" && err == nil {
			log.Dbg("Postgres has been stopped.")
			return nil
		}

		cnt++
		if cnt > 1000 && out != "" && err == nil {
			return fmt.Errorf("Postgres could not be stopped in 15 minutes.")
		}
		if cnt > 900 { // 15 minutes = 900 seconds
			_, err = j.dockerRunCommand("sudo killall -s 9 postgres || true")
		}

		_, err = j.dockerRunCommand("sudo pg_ctlcluster " + j.config.PgVersion + " main stop -m f || true")

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("Cannot stop Postgres. Unreachable code.")
}

// Move pointer to Postgres pgdata to external drive.
func (j *provisionAws) dockerMovePostgresPgData() error {
	err := j.dockerStopPostgres()
	if err != nil {
		return err
	}

	_, err = j.dockerRunCommand("sudo mv /var/lib/postgresql /var/lib/postgresql_original")
	if err != nil {
		return err
	}

	_, err = j.dockerRunCommand("ln -s /storage/postgresql /var/lib/postgresql")
	if err != nil {
		return err
	}

	err = j.dockerStartPostgres()
	if err != nil {
		return err
	}

	return nil
}

// Create ZFS snapshot on drive.
func (j *provisionAws) createZfsSnapshot(name string) error {
	log.Dbg("Create a database snapshot.")

	err := j.dockerStopPostgres()
	if err != nil {
		return err
	}

	out, err := j.ec2ctrl.RunInstanceSshCommand("sudo zfs snapshot -r zpool@"+name, j.config.Debug)
	if err != nil {
		return fmt.Errorf("Cannot create a ZFS snapshot: %s, %v.", out, err)
	}

	err = j.dockerStartPostgres()
	if err != nil {
		return err
	}

	return nil
}

// Rollback to ZFS snapshot on drive.
// TODO(Nikolay): Add comments for all function signatures (desc, params, returning type).
func (j *provisionAws) dockerRollbackZfsSnapshot(name string) error {
	log.Dbg("Rollback the state of the database to the specified snapshot.")

	err := j.dockerStopPostgres()
	if err != nil {
		return err
	}

	out, err := j.ec2ctrl.RunInstanceSshCommand("sudo zfs rollback -f -r zpool@"+name, j.config.Debug)
	if err != nil {
		return fmt.Errorf("Cannot rollback to the ZFS snapshot: %s, %v.", out, err)
	}

	err = j.dockerStartPostgres()
	if err != nil {
		return err
	}

	return nil
}

// Start instance for DB tests.
func (j *provisionAws) startWorkingInstance() error {
	var result bool
	var out string

	err := j.startInstance()
	if err != nil {
		j.stopInstance()
		return fmt.Errorf("Cannot start instance. %v", err)
	}

	// Check instance existance.
	err = j.startInstanceSsh()
	if err != nil {
		j.stopInstance()
		return err
	}

	err = j.attachZfsPancake()
	log.Dbg("Attach ZFS pancake drive:", result, err)
	if err != nil {
		j.stopInstance()
		return fmt.Errorf("Cannot attach the disk. %v", err)
	}

	err = j.startDocker()
	log.Dbg("Start docker:", result, err)
	if err != nil {
		j.stopInstance()
		return fmt.Errorf("Cannot start Docker. %v", err)
	}

	out, err = j.dockerRunCommand("echo 1")
	out = strings.Trim(out, "\n")
	if err != nil || out != "1" {
		j.stopInstance()
		return fmt.Errorf("Cannot get an access to Docker. %v", err)
	}

	err = j.dockerMovePostgresPgData()
	log.Dbg("Move PGdata pointer:", err)
	if err != nil {
		j.stopInstance()
		return fmt.Errorf("Cannot move data to the disk. %v", err)
	}

	return nil
}

// Create DB user.
func (j *provisionAws) dockerCreateDbUser() error {
	var err error
	sql := "select 1 from pg_catalog.pg_roles where rolname = '" + j.config.DbUsername + "'"
	out, err := j.dockerRunCommand("psql -Upostgres -d postgres -t -c \"" + sql + "\"")
	out = strings.Trim(out, "\n ")

	if err != nil {
		return err
	}

	if out == "1" {
		log.Dbg("Test user already exists")
		return nil
	}

	sql = "CREATE ROLE " + j.config.DbUsername + " LOGIN password '" + j.config.DbPassword + "' superuser;"
	out, err = j.dockerRunCommand("psql -Upostgres -d postgres -t -c \""+sql+"\"", false)
	log.Dbg("Create test user", out, err)

	return err
}

// TODO(anatoly): Move to ssh.
func (j *provisionAws) createSshTunnel() error {
	if !SshTunnelExists(j.config.DbUsername, j.config.DbPassword, j.config.Aws.SshTunnelPort) {
		CloseSshTunnel(j.instanceIp)
		err := OpenSshTunnel(j.instanceIp, j.config.Aws.SshTunnelPort, j.config.Aws.Ec2.AwsKeyPath)
		if err != nil {
			return fmt.Errorf("Cannot establish an SSH tunnel: %v", err)
		}
	}

	log.Dbg("SSH tunnel is " + log.GREEN + "ready" + log.END)
	return nil
}
